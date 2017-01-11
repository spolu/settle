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
	"github.com/spolu/settle/mint/lib/plan"
	"github.com/spolu/settle/mint/model"
	"goji.io/pat"
)

const (
	// EndPtSettleTransaction settles a reserved transaction.
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
	Plan *plan.TxPlan
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
	oCtx := ctx

	ctx = db.Begin(oCtx, "mint")
	defer db.LoggedRollback(ctx)

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

	owner := fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		mint.GetHost(ctx))

	if e.Tx.Owner != owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_not_authorized",
			"Only the owner of the transaction can settle it: %s"+
				e.Tx.Owner,
		))
	}

	// Transaction can be either reserved or settled.
	switch e.Tx.Status {
	case mint.TxStCanceled:
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The transaction you are trying to settle is canceled: %s.",
			e.ID,
		))
	case mint.TxStPending:
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The transaction you are trying to settle is pending: %s ",
			e.ID,
		))
	}

	// For canonical settlement we can do away with a shallow plan.
	pl, err := plan.Compute(ctx, e.Client, e.Tx, true)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}
	e.Plan = pl

	// Settle the transaction definitely before we reveal the secret (even if
	// it eventually fails).
	e.Tx.Status = mint.TxStSettled
	err = e.Tx.Save(ctx)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	// At the canonical mint the settlement propagation starts from a virtual
	// Hop which is the length of the plan hops plus one.
	e.Hop = int8(len(e.Plan.Hops))
	e.Secret = *e.Tx.Secret

	err = e.Propagate(ctx)
	if err != nil {
		// If propagation failed we log it and trigger an asyncrhonous one.
		mint.Logf(ctx,
			"Settlement propagation failed: transaction=%s hop=%d error=%s",
			e.ID, e.Hop, err.Error())
		err = async.Queue(ctx,
			task.NewPropagateSettlement(ctx,
				time.Now(), fmt.Sprintf("%s|%d", e.ID, e.Hop)))
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	ctx = db.Begin(oCtx, "mint")
	defer db.LoggedRollback(ctx)

	// Reload the tranasction post propagation.
	tx, err = model.LoadTransactionByID(ctx, e.ID)
	if err != nil || tx == nil {
		return nil, nil, errors.Trace(err) // 500
	}
	e.Tx = tx

	switch e.Tx.Status {
	case mint.TxStSettled:
	default:
		return nil, nil, errors.Newf(
			"Unexpected transaction status %s: %s", e.Tx.Status, e.ID) // 500
	}

	ops, err := model.LoadCanonicalOperationsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	crs, err := model.LoadCanonicalCrossingsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	// Commit the transaction in settled state.
	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			tx, ops, crs,
		)),
	}, nil
}

// ExecutePropagated executes the settlement of a propagated transaction
// (involved mint).
func (e *SettleTransaction) ExecutePropagated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	oCtx := ctx

	ctx = db.Begin(oCtx, "mint")
	defer db.LoggedRollback(ctx)

	tx, err := model.LoadTransactionByID(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if tx == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "transaction_not_found",
			"The transaction you are trying to settle does not exist: %s.",
			e.ID,
		))
	}
	e.Tx = tx

	// We first compute a shallow plan to check whether we should cancel
	pl, err := plan.Compute(ctx, e.Client, e.Tx, true)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}

	if int(e.Hop) >= len(pl.Hops) ||
		pl.Hops[e.Hop].Mint != mint.GetHost(ctx) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The hop provided (%d) does not match the current mint (%s) for "+
				"transaction: %s", e.Hop, mint.GetHost(ctx), e.ID,
		))
	}

	// Check for potential opportunity to cancel before settling.
	if pl.CheckShouldCancel(ctx, e.Client, e.Hop) {
		// Commit the transaction while we cancel.
		db.Commit(ctx)

		// Attempt to trigger a cancelation locally as we know it should
		// cancel. If we fail, just continue as we were.
		_, err := e.Client.CancelTransaction(ctx,
			e.ID, e.Hop, mint.GetHost(ctx))
		if err != nil {
			mint.Logf(ctx,
				"Opportunistic cancellation failed: transaction=%s hop=%d error=%s",
				e.ID, e.Hop, err.Error())
		}

		// Reopen a DB transaction and reload the transaction, hopefully
		// updated.
		ctx = db.Begin(oCtx, "mint")
		defer db.LoggedRollback(ctx)

		tx, err = model.LoadTransactionByID(ctx, e.ID)
		if err != nil || tx == nil {
			return nil, nil, errors.Trace(err) // 500
		}
		e.Tx = tx
	}

	// Transaction can be either reserved or settled.
	switch e.Tx.Status {
	case mint.TxStCanceled:
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The transaction you are trying to settle is canceled: %s.",
			e.ID,
		))
	case mint.TxStPending:
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "settlement_failed",
			"The transaction you are trying to settle is pending: %s ",
			e.ID,
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

	// Compute now the full plan to execute settlement.
	pl, err = plan.Compute(ctx, e.Client, e.Tx, false)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}
	e.Plan = pl

	// Settle will idempotently (at specified hop) settle the
	// transaction (generally called on highest hop first). Subsequent calls (on
	// same hop will be no-ops).
	err = e.Settle(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "settlement_failed",
			"The settlement execution failed for the transaction: %s", e.ID,
		))
	}

	// Mark the transaction as settled (if there's a loop we'll call settle on
	// the other hop even if marked as settled) and store the secret.
	e.Tx.Status = mint.TxStSettled
	e.Tx.Secret = &e.Secret
	err = e.Tx.Save(ctx)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	ops, err := model.LoadCanonicalOperationsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	crs, err := model.LoadCanonicalCrossingsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	// Commit the transaction as well as operations and crossings as settled.
	db.Commit(ctx)

	err = e.Propagate(ctx)
	if err != nil {
		// If propagation failed we log it and trigger an asyncrhonous one.
		mint.Logf(ctx,
			"Settlement propagation failed: transaction=%s hop=%d error=%s",
			e.ID, e.Hop, err.Error())
		err = async.Queue(ctx,
			task.NewPropagateSettlement(ctx,
				time.Now(), fmt.Sprintf("%s|%d", e.ID, e.Hop)))
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	return ptr.Int(http.StatusOK), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			tx, ops, crs,
		)),
	}, nil
}

// Settle settles idempotently the underlying operations and crossings at the
// current hop.
func (e *SettleTransaction) Settle(
	ctx context.Context,
) error {
	if int(e.Hop) >= len(e.Plan.Hops) {
		return errors.Trace(errors.Newf(
			"Hop (%d) is higher than the transaction plan length (%d)",
			e.Hop, len(e.Plan.Hops)))
	}

	h := e.Plan.Hops[e.Hop]
	mint.Logf(ctx,
		"Executing settlement plan: transaction=%s hop=%d", e.ID, e.Hop)

	// Settle the OpAction (should always be defined)
	if h.OpAction != nil {
		op, err := model.LoadCanonicalOperationByTransactionHop(ctx,
			e.ID, e.Hop)
		if err != nil {
			return errors.Trace(err)
		} else if op == nil {
			return errors.Trace(errors.Newf(
				"Operation not found for transaction %s and hop %d",
				e.ID, h))
		}

		if op.Status == mint.TxStSettled {
			mint.Logf(ctx,
				"Skipped operation: id=%s[%s] created=%q propagation=%s "+
					"asset=%s source=%s destination=%s amount=%s "+
					"status=%s transaction=%s",
				op.Owner, op.Token, op.Created, op.Propagation, op.Asset,
				op.Source, op.Destination, (*big.Int)(&op.Amount).String(),
				op.Status, *op.Transaction)

		} else {
			a := h.OpAction

			asset, err := model.LoadCanonicalAssetByName(ctx, *a.OperationAsset)
			if err != nil {
				return errors.Trace(err)
			} else if asset == nil {
				return errors.Trace(errors.Newf(
					"Asset not found: %s", *a.OperationAsset))
			}

			var dstBalance *model.Balance
			if a.OperationDestination != nil &&
				asset.Owner != *a.OperationDestination {
				dstBalance, err =
					model.LoadOrCreateCanonicalBalanceByAssetHolder(ctx,
						asset.Owner, *a.OperationAsset, *a.OperationDestination)
				if err != nil {
					return errors.Trace(err)
				}
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

				err = async.Queue(ctx,
					task.NewPropagateBalance(ctx, time.Now(), dstBalance.ID()))
				if err != nil {
					return errors.Trace(err)
				}
			}

			op.Status = mint.TxStSettled
			err = op.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}

			mint.Logf(ctx,
				"Settled operation: id=%s[%s] created=%q propagation=%s "+
					"asset=%s source=%s destination=%s amount=%s "+
					"status=%s transaction=%s",
				op.Owner, op.Token, op.Created, op.Propagation, op.Asset,
				op.Source, op.Destination, (*big.Int)(&op.Amount).String(),
				op.Status, *op.Transaction)

			opID := op.ID()
			err = async.Queue(ctx,
				task.NewPropagateOperation(ctx, time.Now(), opID))
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	if h.CrAction != nil {
		cr, err := model.LoadCanonicalCrossingByTransactionHop(ctx,
			e.ID, e.Hop)
		if err != nil {
			return errors.Trace(err)
		} else if cr == nil {
			return errors.Trace(errors.Newf(
				"Crossing not found for transaction %s and hop %d",
				e.ID, h))
		}

		if cr.Status == mint.TxStSettled {
			mint.Logf(ctx,
				"Skipped crossing: id=%s[%s] created=%q offer=%s amount=%s "+
					"status=%s transaction=%s",
				cr.Owner, cr.Token, cr.Created, cr.Offer,
				(*big.Int)(&cr.Amount).String(), cr.Status, cr.Transaction)
		} else {
			cr.Status = mint.TxStSettled
			err = cr.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}

			mint.Logf(ctx,
				"Settled crossing: id=%s[%s] created=%q offer=%s amount=%s "+
					"status=%s transaction=%s",
				cr.Owner, cr.Token, cr.Created, cr.Offer,
				(*big.Int)(&cr.Amount).String(), cr.Status, cr.Transaction)
		}
	}

	return nil
}

// Propagate the lock for settlement. Current hop settlement is already
// performed.
func (e *SettleTransaction) Propagate(
	ctx context.Context,
) error {
	if int(e.Hop)-1 >= 0 {
		m := e.Plan.Hops[e.Hop-1].Mint

		mint.Logf(ctx,
			"Propagating settlement: transaction=%s hop=%d mint=%s",
			e.ID, e.Hop, m)

		hop := e.Hop - 1
		_, err := e.Client.SettleTransaction(ctx, e.ID, &hop, &e.Secret, &m)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
