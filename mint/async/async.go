package async

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
)

// Task is the interface for a task.
type Task interface {
	// Name is the name of the task.
	Name() mint.TkName

	// Created is the time of creation of the task.
	Created() time.Time

	// Subject is the subject of the task, generally an object ID.
	Subject() string

	// MaxRetries caps the total number of retries.
	MaxRetries() uint

	// DeadlineforRetry returns the deadline for the provided retry count.
	DeadlineForRetry(retry uint) time.Time

	// Execute idempotently runs the task to completion or errors.
	Execute(ctx context.Context) error
}

// Registrar is used to register task generators within the module. The role of
// the generator for a given mint.TkName is to reconstruct a task from its
// subject, status and retry.
var Registrar = map[mint.TkName](func(
	context.Context,
	time.Time,
	string,
) Task){}

// Deadline represent an execution deadline for task.
type Deadline struct {
	Task  Task
	Model *model.Task
}

// Deadlines is a slice of Deadline implementing sort.Interface
type Deadlines []Deadline

// Len implenents the sort.Interface
func (s Deadlines) Len() int {
	return len(s)
}

// Less implenents the sort.Interface
func (s Deadlines) Less(i, j int) bool {
	return s[i].Deadline().After(s[j].Deadline())
}

// Swap implenents the sort.Interface
func (s Deadlines) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Deadline returns the current deadline for the task.
func (d Deadline) Deadline() time.Time {
	return d.Task.DeadlineForRetry(d.Model.Retry)
}

// Async represents the state of an async queue.
type Async struct {
	Ctx       context.Context
	Pending   Deadlines
	Scheduled chan Deadline

	mutex *sync.Mutex
}

// NewAsync constructs a new async state.
func NewAsync(
	ctx context.Context,
) (*Async, error) {
	a := &Async{
		Ctx:       ctx,
		Pending:   nil,
		Scheduled: make(chan Deadline),
		mutex:     &sync.Mutex{},
	}

	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	tasks, err := model.LoadPendingTasks(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	db.Commit(ctx)

	deadlines := Deadlines{}
	for _, m := range tasks {
		generator, ok := Registrar[m.Name]
		if !ok {
			return nil, errors.Trace(
				errors.Newf("Unregistered task name: %s", m.Name))
		}

		d := Deadline{
			Task:  generator(ctx, m.Created, m.Subject),
			Model: m,
		}
		deadlines = append(deadlines, d)
		mint.Logf(ctx, "Retrieved task: "+
			"name=%s subject=%s retry=%d deadline=%q",
			d.Task.Name(), d.Task.Subject(), d.Model.Retry, d.Deadline())
	}

	a.Pending = deadlines
	sort.Sort(a.Pending)

	a.schedule(ctx)

	return a, nil
}

// schedule attempts to schedule an eligible task in a non blocking way. If
// there is no task to schedule or the Scheduled channel is blocked, it's a
// no-op. Can be called as often as needed.
// a.mutex must be held.
func (a *Async) schedule(
	ctx context.Context,
) {
	if len(a.Pending) == 0 {
		return
	}
	d := a.Pending[len(a.Pending)-1]
	if d.Deadline().Before(time.Now()) {
		select {
		case a.Scheduled <- d:
			a.Pending = a.Pending[:len(a.Pending)-1]

			mint.Logf(ctx, "Scheduled task: "+
				"name=%s subject=%s retry=%d deadline=%s",
				d.Task.Name(), d.Task.Subject(), d.Model.Retry,
				d.Deadline().String())
		default:
		}
	} else {
		mint.Logf(ctx, "Scheduler next task: "+
			"name=%s subject=%s retry=%d deadline=%s duration=%s",
			d.Task.Name(), d.Task.Subject(), d.Model.Retry,
			d.Deadline().String(), d.Deadline().Sub(time.Now()).String())
	}

}

// Queue queues a new task by adding it to the list of pending tasks and
// calling Schedule. Queue does not begin a new transaction as it is meant to
// be called within a transaction block (offer creation, operation creation,
// ...).
func (a *Async) Queue(
	ctx context.Context,
	t Task,
) error {
	m, err := model.CreateTask(ctx,
		t.Created(),
		t.Name(),
		t.Subject(),
		mint.TkStPending,
		0,
	)
	if err != nil {
		return errors.Trace(err)
	}

	a.AppendAndSchedule(ctx, Deadline{
		Task:  t,
		Model: m,
	})

	return nil
}

// LockAndSchedule locks and schedule a task if possible
func (a *Async) LockAndSchedule(
	ctx context.Context,
) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	sort.Sort(a.Pending)

	a.schedule(ctx)
}

// AppendAndSchedule appends a deadline to the list of pending deadlines while
// preserving its order and calls schedule.
func (a *Async) AppendAndSchedule(
	ctx context.Context,
	d Deadline,
) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.Pending = append(a.Pending, d)
	sort.Sort(a.Pending)

	mint.Logf(ctx, "Queued task: "+
		"name=%s subject=%s retry=%d deadline=%q",
		d.Task.Name(), d.Task.Subject(), d.Model.Retry, d.Deadline())

	a.schedule(ctx)
}

// RunOne runs the specified deadline and re-add it to the list of pending
// deadline if it fails.
func (a *Async) RunOne(
	d Deadline,
) {
	mint.Logf(a.Ctx, "Executing task: "+
		"name=%s subject=%s retry=%d deadline=%q",
		d.Task.Name(), d.Task.Subject(), d.Model.Retry, d.Deadline())

	err := d.Task.Execute(With(a.Ctx, a))

	ctx := db.Begin(a.Ctx, "mint")
	defer db.LoggedRollback(ctx)

	if err != nil {
		mint.Logf(ctx, "Error executing task: "+
			"name=%s subject=%s retry=%d error=%s",
			d.Task.Name(), d.Task.Subject(), d.Model.Retry, err.Error())
		for _, line := range errors.ErrorStack(err) {
			mint.Logf(ctx, "  %s", line)
		}

		d.Model.Retry++
		if d.Model.Retry > d.Task.MaxRetries() {
			d.Model.Status = mint.TkStFailed
		}
	} else {
		mint.Logf(ctx, "Successfuly executed task: "+
			"name=%s subject=%s retry=%d",
			d.Task.Name(), d.Task.Subject(), d.Model.Retry)

		d.Model.Status = mint.TkStSucceeded
	}

	err = d.Model.Save(ctx)
	if err != nil {
		mint.Logf(ctx, "Error saving task: "+
			"name=%s subject=%s retry=%d error=%s",
			d.Task.Name(), d.Task.Subject(), d.Model.Retry, err.Error())
	}

	db.Commit(ctx)

	if d.Model.Status == mint.TkStPending {
		go func() {
			a.AppendAndSchedule(ctx, d)
		}()
	} else {
		go func() {
			time.Sleep(100 * time.Millisecond)
			a.LockAndSchedule(a.Ctx)
		}()
	}
}

// Run should be called from a go routine to execute task as a worker. Multiple
// worker can be run concurrently.
func (a *Async) Run() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			a.LockAndSchedule(a.Ctx)
		}
	}()
	for d := range a.Scheduled {
		a.RunOne(d)
	}
}

// ContextKey is the type of the key used with context to carry contextual
// async state.
type ContextKey string

const (
	// asyncKey the context.Context key to store the async state.
	asyncKey ContextKey = "async.async"
)

// With stores the async state in the provided context.
func With(
	ctx context.Context,
	async *Async,
) context.Context {
	return context.WithValue(ctx, asyncKey, async)
}

// Get returns the async state currently stored in the context.
func Get(
	ctx context.Context,
) *Async {
	return ctx.Value(asyncKey).(*Async)
}

// Queue queues a task for execution by the async queue.
func Queue(
	ctx context.Context,
	t Task,
) error {
	async := Get(ctx)
	return async.Queue(ctx, t)
}

// TestRunOne runs one task off of the list of pending tasks, In tests we don't
// have any worker so we use this ot run tasks syncrhonously as needed.
func TestRunOne(
	ctx context.Context,
) {
	a := Get(ctx)
	var d Deadline

	a.mutex.Lock()

	if len(a.Pending) == 0 {
		a.mutex.Unlock()
		return
	}
	d, a.Pending = a.Pending[len(a.Pending)-1], a.Pending[:len(a.Pending)-1]

	a.mutex.Unlock()

	a.RunOne(d)
}
