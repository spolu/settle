package task

import (
	"context"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/lib/plan"
	"github.com/spolu/settle/mint/model"
)

const (
	// TkExpireTransaction expires a transaction
	TkExpireTransaction mint.TkName = "ExpireTransaction"
)

func init() {
	async.Registrar[TkExpireTransaction] = NewExpireTransaction
}

// ExpireTransaction is in charge of attempting to cancel the transcation (if
// possible) mint.TransactionExpiryMs after creation.
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
	expiry := time.Duration(mint.TransactionExpiryMs) * time.Millisecond
	return t.Created().Add(expiry + time.Duration(retry)*expiry)
}

// Execute idempotently runs the task to completion or errors.
func (t *ExpireTransaction) Execute(
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

	if tx.Status == mint.TxStSettled {
		mint.Logf(ctx,
			"Skipping settled transaction expiry: transaction=%s status=%s",
			tx.ID(), tx.Status)
		return nil
	}

	// For expiration we can do away with a shallow plan.
	plan, err := plan.Compute(ctx, client, tx, true)
	if err != nil {
		return errors.Trace(err)
	}

	// Retrieve our maximal hop for this mint.
	_, maxHop, err := plan.MinMaxHop(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	hop := *maxHop

	_, err = client.CancelTransaction(ctx, tx.ID(), hop, mint.GetHost(ctx))
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
