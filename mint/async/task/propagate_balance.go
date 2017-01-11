package task

import (
	"context"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/model"
)

const (
	// TkPropagateBalance propagates an operation
	TkPropagateBalance mint.TkName = "PropagateBalance"
)

func init() {
	async.Registrar[TkPropagateBalance] = NewPropagateBalance
}

// PropagateBalance is in charge of propagating the balance to the holder mint
// if applicable.
type PropagateBalance struct {
	created time.Time
	id      string
}

// NewPropagateBalance constructs and initializes the task.
func NewPropagateBalance(
	ctx context.Context,
	created time.Time,
	subject string,
) async.Task {
	return &PropagateBalance{
		created: created,
		id:      subject,
	}
}

// Name returns the task name.
func (t *PropagateBalance) Name() mint.TkName {
	return TkPropagateBalance
}

// Created returns the task creation time.
func (t *PropagateBalance) Created() time.Time {
	return t.created
}

// Subject returns the task subject.
func (t *PropagateBalance) Subject() string {
	return t.id
}

// MaxRetries returns the max retries for the task.
func (t *PropagateBalance) MaxRetries() uint {
	return 18
}

// DeadlineForRetry returns the deadline for the provided retry count.
func (t *PropagateBalance) DeadlineForRetry(
	retry uint,
) time.Time {
	return t.Created().Add((1<<retry - 1) * time.Second)
}

// Execute idempotently runs the task to completion or errors.
func (t *PropagateBalance) Execute(
	ctx context.Context,
) error {
	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	balance, err := model.LoadCanonicalBalanceByID(ctx, t.id)
	if err != nil {
		return errors.Trace(err)
	} else if balance == nil {
		return errors.Trace(errors.Newf(
			"Canonical balance not found: %s", t.id))
	}

	db.Commit(ctx)

	_, host, err := mint.UsernameAndMintHostFromAddress(ctx, balance.Holder)
	if err != nil {
		return errors.Trace(err)
	}

	if host != mint.GetHost(ctx) {
		_, err := client.PropagateBalance(ctx, balance.ID(), host)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
