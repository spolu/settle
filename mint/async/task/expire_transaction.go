// OWNER: stan

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
	// TkExpireTransaction expires a transaction
	TkExpireTransaction mint.TkName = "ExpireTransaction"
)

func init() {
	async.Registrar[TkExpireTransaction] = NewExpireTransaction
}

// ExpireTransaction is in charge of expiring transactions 1h after their
// creation in case they haven't been settled.
type ExpireTransaction struct {
	created time.Time
	id      string
}

// NewExpireTransaction constructs and initializes the task.
func NewExpireTransaction(
	ctx context.Context,
	created time.Time,
	subject string,
) async.Task {
	return &ExpireTransaction{
		created: created,
		id:      subject,
	}
}

// Name returns the task name.
func (t *ExpireTransaction) Name() mint.TkName {
	return TkExpireTransaction
}

// Created returns the task creation time.
func (t *ExpireTransaction) Created() time.Time {
	return t.created
}

// Subject returns the task subject.
func (t *ExpireTransaction) Subject() string {
	return t.id
}

// MaxRetries returns the max retries for the task.
func (t *ExpireTransaction) MaxRetries() uint {
	return 8
}

// DeadlineForRetry returns the deadline for the provided retry count.
func (t *ExpireTransaction) DeadlineForRetry(
	retry uint,
) time.Time {
	return t.Created().Add(time.Duration(retry+1) * time.Hour)
}

// Execute idempotently runs the task to completion or errors.
func (t *ExpireTransaction) Execute(
	ctx context.Context,
) error {
	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	txn, err := model.LoadTransactionByID(ctx, t.id)
	if err != nil {
		return errors.Trace(err)
	}
	ops, err := model.LoadCanonicalOperationsByTransaction(ctx, t.id)
	if err != nil {
		return errors.Trace(err)
	}
	crs, err := model.LoadCrossingsByTransaction(ctx, t.id)
	if err != nil {
		return errors.Trace(err)
	}

	if txn != nil {
		txn.Status = mint.TxStCanceled
		err = txn.Save(ctx)
		if err != nil {
			return errors.Trace(err)
		}
	}
	for _, op := range ops {
		op.Status = mint.TxStCanceled
		err = op.Save(ctx)
		if err != nil {
			return errors.Trace(err)
		}
	}
	for _, cr := range crs {
		cr.Status = mint.TxStCanceled
		err = cr.Save(ctx)
		if err != nil {
			return errors.Trace(err)
		}
	}

	db.Commit(ctx)

	return nil
}
