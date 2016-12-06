// OWNER: stan

package async

import (
	"context"
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

	// Subject is the subject of the task, generally an object ID.
	Subject() string

	// Status is the status of the task.
	Status() mint.TkStatus

	// Execute idempotently runs the task to completion or errors.
	Execute(ctx context.Context) error

	// MaxRetries caps the total number of retries.
	MaxRetries() uint64

	// DeadlineforRetry returns the deadline for the provided retry count.
	DeadlineForRetry(retry uint64) time.Time
}

// registrar is used to register task generators within the module. The role of
// the generator for a given mint.TkName is to reconstruct a task from its
// subject, status and retry.
var registrar = map[mint.TkName](func(
	context.Context,
	string,
	mint.TkStatus,
	uint64,
) Task){}

// Deadline represent an execution deadline for task.
type Deadline struct {
	Task  Task
	Model *model.Task
}

// Deadline returns the current deadline for the task.
func (d Deadline) Deadline() time.Time {
	return d.Task.DeadlineForRetry(d.Model.Retry)
}

// Async represents the state of an async queue.
type Async struct {
	Ctx       context.Context
	Pending   []Deadline
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
		Scheduled: make(chan Deadline, 1),
		mutex:     &sync.Mutex{},
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	tasks, err := model.LoadPendingTasks(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	db.Commit(ctx)

	deadlines := []Deadline{}
	for _, m := range tasks {
		generator, ok := registrar[m.Name]
		if !ok {
			return nil, errors.Trace(
				errors.Newf("Unregistered task name: %s", m.Name))
		}
		t := generator(ctx,
			m.Subject,
			m.Status,
			m.Retry,
		)
		deadlines = append(deadlines, Deadline{
			Task:  t,
			Model: m,
		})
	}

	a.Pending = deadlines
	// TODO(stan): sort a.Deadlines by descending task.Deadline()

	a.schedule()

	return a, nil
}

// schedule attempts to schedule an eligible task in a non blocking way. If
// there is no task to schedule or the Scheduled channel is blocked, it's a
// no-op. Can be called as often as needed.
// a.mutex must be held.
func (a *Async) schedule() {
	if len(a.Pending) == 0 {
		return
	}
	d := a.Pending[len(a.Pending)-1]
	if d.Deadline().After(time.Now()) {
		select {
		case a.Scheduled <- d:
			a.Pending = a.Pending[:len(a.Pending)-1]
		}
	}
}

// Queue queues a new task by adding it to the list of pending tasks and
// calling Schedule.
func (a *Async) Queue(
	t Task,
) error {
	ctx := db.Begin(a.Ctx)
	defer db.LoggedRollback(ctx)

	m, err := model.CreateTask(ctx,
		t.Name(),
		t.Subject(),
		mint.TkStPending,
		0,
	)
	if err != nil {
		return errors.Trace(err)
	}

	db.Commit(ctx)

	a.AppendAndSchedule(Deadline{
		Task:  t,
		Model: m,
	})

	return nil
}

// AppendAndSchedule appends a deadline to the list of pending deadlines while
// preserving its order and calls schedule.
func (a *Async) AppendAndSchedule(
	d Deadline,
) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.Pending = append(a.Pending, d)
	// TODO(stan): sort a.Deadlines by descending task.Deadline()

	a.schedule()
}

// RunOne runs the specified deadline and re-add it to the list of pending
// deadline if it fails.
func (a *Async) RunOne(
	d Deadline,
) {
	err := d.Task.Execute(a.Ctx)

	ctx := db.Begin(a.Ctx)
	defer db.LoggedRollback(ctx)

	if err != nil {
		mint.Logf(ctx, "Error executing task: "+
			"name=%s subject=%s retry=%d error=%s",
			d.Task.Name(), d.Task.Subject(), d.Model.Retry, err.Error())

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
		a.AppendAndSchedule(d)
	}
}

// Run should be called from a go routine to execute task as a worker. Multiple
// worker can be run concurrently.
func (a *Async) Run() {
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
	return async.Queue(t)
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
		return
	}
	d, a.Pending = a.Pending[len(a.Pending)-1], a.Pending[:len(a.Pending)-1]

	a.mutex.Unlock()

	a.RunOne(d)
}
