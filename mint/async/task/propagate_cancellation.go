package task

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/lib/plan"
	"github.com/spolu/settle/mint/model"
)

const (
	// TkPropagateCancellation propagates a transaction cancellation.
	TkPropagateCancellation mint.TkName = "PropagateCancellation"
)

func init() {
	async.Registrar[TkPropagateCancellation] = NewPropagateCancellation
}

// PropagateCancellation is in charge of propagating the cancellation of a
// tranasaction asyncrhonously if synchronous propagation failed.
type PropagateCancellation struct {
	created time.Time
	id      string
	hop     int8
}

// NewPropagateCancellation constructs and initializes the task.
func NewPropagateCancellation(
	ctx context.Context,
	created time.Time,
	subject string,
) async.Task {
	ss := strings.Split(subject, "|")
	if len(ss) != 2 {
		panic(errors.Newf("Invalid subject: %s", subject))
	}
	h, err := strconv.ParseInt(ss[1], 10, 8)
	if err != nil {
		panic(err)
	}
	hop := int8(h)

	return &PropagateCancellation{
		created: created,
		id:      ss[0],
		hop:     hop,
	}
}

// Name returns the task name.
func (t *PropagateCancellation) Name() mint.TkName {
	return TkPropagateCancellation
}

// Created returns the task creation time.
func (t *PropagateCancellation) Created() time.Time {
	return t.created
}

// Subject returns the task subject.
func (t *PropagateCancellation) Subject() string {
	return t.id
}

// MaxRetries returns the max retries for the task.
func (t *PropagateCancellation) MaxRetries() uint {
	return 18
}

// DeadlineForRetry returns the deadline for the provided retry count.
func (t *PropagateCancellation) DeadlineForRetry(
	retry uint,
) time.Time {
	return t.Created().Add((1<<retry - 1) * time.Second)
}

// Execute idempotently runs the task to completion or errors.
func (t *PropagateCancellation) Execute(
	ctx context.Context,
) error {
	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	tx, err := model.LoadTransactionByID(ctx, t.id)
	if err != nil {
		return errors.Trace(err)
	} else if tx == nil {
		return errors.Trace(
			errors.Newf("Transaction not found: %s", t.id))
	}

	db.Commit(ctx)

	// For propagation we can do away with a shallow plan.
	plan, err := plan.Compute(ctx, client, tx, true)
	if err != nil {
		return errors.Trace(err)
	}

	if int(t.hop)-1 >= 0 {
		m := plan.Hops[t.hop-1].Mint

		mint.Logf(ctx,
			"Propagating cancellation: transaction=%s hop=%d mint=%s",
			tx.ID(), t.hop, m)

		_, err := client.CancelTransaction(ctx, tx.ID(), t.hop-1, m)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
