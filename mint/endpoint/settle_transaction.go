// OWNER: stan

package endpoint

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"golang.org/x/crypto/scrypt"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/async/task"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
	"goji.io/pat"
)

const (
	// EndPtSettleTransaction creates a new transaction.
	EndPtSettleTransaction EndPtName = "SettleTransaction"
)

func init() {
	registrar[EndPtSettleTransaction] = NewSettleTransaction
}

// SettleTransaction creates a new transaction.
type SettleTransaction struct {
	Client *mint.Client

	// Parameters
	Hop    int8
	ID     string
	Token  string
	Owner  string
	Secret string

	// State
	Tx   *model.Transaction
	Plan *TxPlan
}

// NewSettleTransaction constructs and initialiezes the endpoint.
func NewSettleTransaction(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}
	return &SettleTransaction{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *SettleTransaction) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	switch authentication.Get(ctx).Status {
	case authentication.AutStSkipped:
		// Validate hop.
		hop, err := ValidateHop(ctx, r.PostFormValue("hop"))
		if err != nil {
			return errors.Trace(err)
		}
		e.Hop = *hop

		// Validate secret.
		secret, err := ValidateSecret(ctx, r.PostFormValue("secret"))
		if err != nil {
			return errors.Trace(err)
		}
		e.Secret = *secret
	}

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "transaction"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Token = *token
	e.Owner = *owner

	return nil
}

// Execute executes the endpoint.
func (e *SettleTransaction) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	switch authentication.Get(ctx).Status {
	case authentication.AutStSkipped:
		return e.ExecutePropagated(ctx)
	case authentication.AutStSucceeded:
		return e.ExecuteCanonical(ctx)
	}
	return nil, nil, errors.Trace(errors.Newf(
		"Authentication status not expected: %s",
		authentication.Get(ctx).Status))
}

// ExecuteCanonical executes the canonical settlement of a transaction (owner
// mint).
func (e *SettleTransaction) ExecuteCanonical(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	// No need to lock the transaction here as we are the only mint to know its
	// secret before it propagates (also, settlement propagates back to us).
	txStore.Init(ctx, e.ID)

	tx, err := model.LoadCanonicalTransactionByOwnerToken(ctx,
		e.Owner, e.Token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if tx == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "transaction_not_found",
			"The transaction you are trying to settle does not "+
				"exist: %s.", e.ID,
		))
	}
	e.Tx = tx

	if e.Tx.Expiry.Before(time.Now()) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The transaction you are trying to settle has expired: %s. It "+
				"will automatically be canceled within an hour of its "+
				"creation (created at %s).", e.ID,
			e.Tx.Created.UTC().Format(time.UnixDate),
		))
	}
	if e.Tx.Status == mint.TxStCanceled {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The transaction you are trying to settle is already "+
				"canceled: %s.", e.ID,
		))
	}

	// Store the transcation in memory so that it's latest version is available
	// through API before we commit.
	txStore.Store(ctx, e.ID, tx)
	defer txStore.Clear(ctx, e.ID)

	e.Plan = txStore.GetPlan(ctx, e.ID)
	if e.Plan == nil {
		plan, err := ComputePlan(ctx, e.Client, e.Tx)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "settlement_failed",
				"The plan computation for the transaction failed: %s", e.ID,
			))
		}
		txStore.StorePlan(ctx, e.ID, plan)
		e.Plan = plan
	}

	// Set the Hop to the the length of the plan to call Settle
	e.Hop = int8(len(e.Plan.Actions))
	e.Secret = *tx.Secret

	err = e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The transaction failed to settle on required mints: %s.",
			e.ID,
		))
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			txStore.Get(ctx, e.ID),
			txStore.GetOperations(ctx, e.ID),
			txStore.GetCrossings(ctx, e.ID),
		)),
	}, nil
}

// ExecutePropagated executes the settlement of a propagated transaction
// (involved mint).
func (e *SettleTransaction) ExecutePropagated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	txStore.Init(ctx, e.ID)

	// This is used to be sure to unlock only once even if we use defer and
	// unlock explicitely before propagation.
	u := false
	unlock := func() {
		if !u {
			u = true
			txStore.Unlock(ctx, e.ID)
		}
	}
	lock := func() {
		u = false
		txStore.Lock(ctx, e.ID)
	}

	lock()
	defer unlock()

	mustCommit := false
	if txStore.Get(ctx, e.ID) != nil {
		// If we find the transaction in the txStore, we also reuse the
		// underlying db.Transaction so that the whole settlement is run on
		// one single underlying db.Transaction (more consistent and avoids
		// locking issues).
		dbTx := txStore.GetDBTransaction(ctx, e.ID)
		ctx = db.WithTransaction(ctx, *dbTx)
		mint.Logf(ctx, "Transaction: reuse %s", dbTx.Token)

		e.Tx = txStore.Get(ctx, e.ID)
	} else {
		mustCommit = true
		ctx = db.Begin(ctx)
		defer db.LoggedRollback(ctx)

		tx, err := model.LoadPropagatedTransactionByOwnerToken(ctx,
			e.Owner, e.Token)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		} else if tx == nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
				404, "transaction_not_found",
				"The transaction you are trying to settle does not "+
					"exist: %s.", e.ID,
			))
		}
		e.Tx = tx

		// If the transaction is already settled in database, we must return
		// 200 here to avoid double settlement (also if we're settled, we
		// already funded the money and are convinced that we were funded as
		// well).
		if e.Tx.Status == mint.TxStSettled {
			mint.Logf(ctx,
				"Transaction already settled: transaction=%s hop=%d",
				e.ID, e.Hop)
			ops, err := model.LoadCanonicalOperationsByTransaction(ctx, e.ID)
			if err != nil {
				return nil, nil, errors.Trace(err) // 500
			}
			crs, err := model.LoadCrossingsByTransaction(ctx, e.ID)
			if err != nil {
				return nil, nil, errors.Trace(err) // 500
			}
			return ptr.Int(http.StatusOK), &svc.Resp{
				"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
					e.Tx, ops, crs)),
			}, nil
		} else if e.Tx.Status == mint.TxStCanceled {
			mint.Logf(ctx,
				"Transaction already canceled: transaction=%s hop=%d",
				e.ID, e.Hop)
			return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
				402, "settlement_failed",
				"The transaction you are trying to settle is already "+
					"canceled: %s.", e.ID,
			))
		}

		// Store the transcation in memory so that it's latest version is available
		// through API before we commit.
		txStore.Store(ctx, e.ID, tx)
		defer txStore.Clear(ctx, e.ID)
	}

	if e.Tx.Expiry.Before(time.Now()) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The transaction you are trying to settle has expired: %s. It "+
				"will automatically be canceled within an hour of its "+
				"creation (created at %s).", e.ID,
			e.Tx.Created.UTC().Format(time.UnixDate),
		))
	}

	h, err := scrypt.Key([]byte(e.Secret), []byte(e.Tx.Token), 16384, 8, 1, 64)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}
	if e.Tx.Lock != base64.StdEncoding.EncodeToString(h) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The secret provided does not match the lock value for "+
				"transaction: %s", e.ID,
		))
	}

	e.Plan = txStore.GetPlan(ctx, e.ID)
	if e.Plan == nil {
		plan, err := ComputePlan(ctx, e.Client, e.Tx)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "settlement_failed",
				"The plan computation for the transaction failed: %s", e.ID,
			))
		}
		txStore.StorePlan(ctx, e.ID, plan)
		e.Plan = plan
	}

	if e.Plan.Actions[e.Hop].Mint != mint.GetHost(ctx) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The hop provided does not match the current mint for "+
				"transaction: %s", e.ID,
		))
	}

	// We unlock the tranaction before propagating.
	unlock()

	err = e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The transaction failed to settle on required mints: %s.",
			e.ID,
		))
	}

	// Reacquire the lock for final settlement.
	lock()

	// Settle will idempotently (across tranasction) settle the whole
	// transaction (generally called on lowest hop first). Subsequent calls (on
	// higher hops will be no-ops). It is fine to settle the whole transaction
	// here as we effectively trust other mints will dtrt and we know we have
	// at least communicated the lock to everyone here.
	// In case of cycle we are protected by the shared transaction on that
	// mint, meaning that a higher mint failing would fail the entire
	// settlement as well.
	err = e.Settle(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The settlement execution failed for the transaction: %s", e.ID,
		))
	}

	if mustCommit {
		db.Commit(ctx)
	}

	return ptr.Int(http.StatusOK), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			txStore.Get(ctx, e.ID),
			txStore.GetOperations(ctx, e.ID),
			txStore.GetCrossings(ctx, e.ID),
		)),
	}, nil
}

// Settle checks the secret against the lock and settles the underlying
// operation or crossing.
func (e *SettleTransaction) Settle(
	ctx context.Context,
) error {
	// Simply return if we already settled.
	if e.Tx.Status == mint.TxStSettled {
		return nil
	}

	for h := 0; h < len(e.Plan.Actions); h++ {
		if e.Plan.Actions[h].Mint != mint.GetHost(ctx) {
			continue
		}

		// Defense in depth as we should go only once through this.
		a := e.Plan.Actions[h]
		if a.IsExecuted {
			mint.Logf(ctx,
				"Skipping settlement plan: transaction=%s hop=%d", e.ID, h)
			return nil
		}
		mint.Logf(ctx,
			"Executing settlement plan: transaction=%s hop=%d", e.ID, h)

		// We have the transaction lock so this is safe to write. Also we can mark
		// it as executed right away since everything gets canceled in case of
		// error.
		a.IsExecuted = true

		switch a.Type {
		case TxActTpOperation:

			asset, err := model.LoadAssetByName(ctx, *a.OperationAsset)
			if err != nil {
				return errors.Trace(err)
			} else if asset == nil {
				return errors.Trace(errors.Newf(
					"Asset not found: %s", *a.OperationAsset))
			}

			var dstBalance *model.Balance
			if a.OperationDestination != nil &&
				asset.Owner != *a.OperationDestination {
				dstBalance, err = model.LoadOrCreateBalanceByAssetHolder(ctx,
					asset.Owner, *a.OperationAsset, *a.OperationDestination)
				if err != nil {
					return errors.Trace(err)
				}
			}

			op, err := model.LoadCanonicalOperationByTransactionHop(ctx,
				e.ID, int8(h))
			if err != nil {
				return errors.Trace(err)
			} else if op == nil {
				return errors.Trace(errors.Newf(
					"Operation not found for transaction %s and hop %d",
					e.ID, h))
			}

			if dstBalance != nil {
				(*big.Int)(&dstBalance.Value).Add(
					(*big.Int)(&dstBalance.Value), (*big.Int)(&op.Amount))
				// Checks if the dstBalance is positive and not overflown.
				b := (*big.Int)(&dstBalance.Value)
				if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
					b.Cmp(new(big.Int)) < 0 {
					return errors.Trace(errors.Newf(
						"Invalid resulting balance: %s", b.String()))
				}

				err = dstBalance.Save(ctx)
				if err != nil {
					return errors.Trace(err)
				}
			}

			op.Status = mint.TxStSettled
			err = op.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}

			// Store the operation so that it's available when the transaction is
			// returned from this mint.
			txStore.StoreOperation(ctx, e.ID, op)

			mint.Logf(ctx,
				"Settled operation: id=%s[%s] created=%q propagation=%s "+
					"asset=%s source=%s destination=%s amount=%s "+
					"status=%s transaction=%s",
				op.Owner, op.Token, op.Created, op.Propagation, op.Asset,
				op.Source, op.Destination, (*big.Int)(&op.Amount).String(),
				op.Status, *op.Transaction)

			opID := fmt.Sprintf("%s[%s]", op.Owner, op.Token)
			err = async.Queue(ctx, task.NewPropagateOperation(ctx,
				time.Now(), opID))
			if err != nil {
				return errors.Trace(err)
			}

		case TxActTpCrossing:

			cr, err := model.LoadCrossingByTransactionHop(ctx, e.ID, int8(h))
			if err != nil {
				return errors.Trace(err)
			} else if cr == nil {
				return errors.Trace(errors.Newf(
					"Crossing not found for transaction %s and hop %d",
					e.ID, h))
			}

			cr.Status = mint.TxStSettled
			err = cr.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}

			// Store the crossing so that it's available when the transaction is
			// returned from this mint.
			txStore.StoreCrossing(ctx, e.ID, cr)

			mint.Logf(ctx,
				"Settled crossing: id=%s[%s] created=%q offer=%s amount=%s "+
					"status=%s transaction=%s",
				cr.Owner, cr.Token, cr.Created, cr.Offer,
				(*big.Int)(&cr.Amount).String(), cr.Status, cr.Transaction)
		}
	}

	e.Tx.Status = mint.TxStSettled
	err := e.Tx.Save(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Propagate recursively settles from the last mint to the canonical one.
func (e *SettleTransaction) Propagate(
	ctx context.Context,
) error {
	if e.Hop-1 >= 0 {
		h := e.Hop - 1
		m := e.Plan.Actions[h].Mint

		mint.Logf(ctx,
			"Propagating settlement: transaction=%s hop=%d mint=%s",
			e.ID, e.Hop, m)

		_, err := e.Client.SettleTransaction(ctx,
			e.ID, &h, &e.Secret, &m)
		if err != nil {
			return errors.Trace(err)
		}
	}

	// If e.Hop == 0 we return no error, as we are the canonical mint and we
	// are not depending on any other mint for settlement.
	// Otherwise, if there was no error, we can trust that the mint before us
	// in the plan has settled the action we depend on.

	return nil
}
