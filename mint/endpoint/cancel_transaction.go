package endpoint

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"time"

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
	// EndPtCancelTransaction cancels a reserved transaction.
	EndPtCancelTransaction EndPtName = "CancelTransaction"
)

func init() {
	registrar[EndPtCancelTransaction] = NewCancelTransaction
}

// CancelTransaction creates a new transaction.
type CancelTransaction struct {
	Client *mint.Client

	// Parameters
	Hop   int8
	ID    string
	Token string
	Owner string

	// State
	Tx   *model.Transaction
	Plan *plan.TxPlan
}

// NewCancelTransaction constructs and initialiezes the endpoint.
func NewCancelTransaction(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}
	return &CancelTransaction{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *CancelTransaction) Validate(
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
func (e *CancelTransaction) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	switch authentication.Get(ctx).Status {
	case authentication.AutStSkipped:
		return e.ExecutePropagated(ctx)
	case authentication.AutStSucceeded:
		return e.ExecuteAuthenticated(ctx)
	}
	return nil, nil, errors.Trace(errors.Newf(
		"Authentication status not expected: %s",
		authentication.Get(ctx).Status))
}

// ExecuteAuthenticated executes the authenticated cancellation of a
// transaction.
func (e *CancelTransaction) ExecuteAuthenticated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	// We load the transaction even if it has been propagated as the only node
	// than can really trigger a cancellation is the last node of the
	// transaction plan.
	tx, err := model.LoadTransactionByID(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if tx == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "transaction_not_found",
			"The transaction you are trying to cancel does not exist: %s.",
			e.ID,
		))
	}
	e.Tx = tx

	// Transaction can be either pending or reserved.
	switch e.Tx.Status {
	case mint.TxStSettled:
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "cancellation_failed",
			"The transaction you are trying to cancel is settled: %s.",
			e.ID,
		))
	}

	pl, err := plan.Compute(ctx, e.Client, e.Tx, false)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "cancellation_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}
	e.Plan = pl

	minHop, maxHop, err := e.Plan.MinMaxHop(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "cancellation_failed",
			"This node is not part of the transaction plan for %s", e.ID,
		))
	}
	e.Hop = *maxHop

	owner := fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		mint.GetHost(ctx))

	// Check that the requestor is the owner of the transaction or the
	// operation associated with the transaction at this hop.
	if owner != e.Tx.Owner && owner != e.Plan.Hops[e.Hop].OpAction.Owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "cancellation_not_authorized",
			"Only the owner of the action associated with the highest hop of "+
				"this mint can cancel the transaction: %s",
			e.Plan.Hops[e.Hop].OpAction.Owner,
		))
	}

	// Check cancelation can be performed (either we're the last node, or the
	// node after us has already canceled the transaction, or the node after us
	// does not know about the transaction).
	if !e.Plan.CheckCanCancel(ctx, e.Client, e.Hop) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "cancellation_failed",
			"This transaction has not been cancelled by the next node on the "+
				"transaction plan: %s",
			e.Plan.Hops[e.Hop+1].Mint,
		))
	}

	// Cancel will idempotently cancel the transaction on all hops that are
	// involving this mint.
	err = e.Cancel(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "cancellation_failed",
			"The cancellation execution failed for the transaction: %s", e.ID,
		))
	}

	// We mark the transaction as cancelled if the hop of this is the minimal
	// one for this mint. Cancelation checks only use operations and crossings
	// so the status of a transaction is mostly indicative, but we want to mark
	// it as cancelled only after it is cancelled at all hops.
	if e.Hop == *minHop {
		e.Tx.Status = mint.TxStCanceled
		err = e.Tx.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	ops, err := model.LoadCanonicalOperationsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	crs, err := model.LoadCanonicalCrossingsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	err = e.Propagate(ctx)
	if err != nil {
		// If cancellation propagation failed we log it and trigger an
		// asyncrhonous one. In any case the node before us will check on us
		// before attempting to settle as well.
		mint.Logf(ctx,
			"Cancellation propagation failed: transaction=%s hop=%d error=%s",
			e.ID, e.Hop, err.Error())
		err = async.Queue(ctx,
			task.NewPropagateCancellation(ctx,
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

// ExecutePropagated executes the settlement of a propagated transaction
// (involved mint).
func (e *CancelTransaction) ExecutePropagated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
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

	// Transaction can be either pending, reserved or already canceled.
	switch e.Tx.Status {
	case mint.TxStSettled:
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "cancellation_failed",
			"The transaction you are trying to cancel is settled: %s.",
			e.ID,
		))
	}

	pl, err := plan.Compute(ctx, e.Client, e.Tx, false)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "cancellation_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}
	e.Plan = pl

	minHop, _, err := e.Plan.MinMaxHop(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "cancellation_failed",
			"This node is not part of the transaction plan for %s", e.ID,
		))
	}

	if int(e.Hop) >= len(e.Plan.Hops) ||
		e.Plan.Hops[e.Hop].Mint != mint.GetHost(ctx) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "cancellation_failed",
			"The hop provided (%d) does not match the current mint (%s) for "+
				"transaction: %s", e.Hop, mint.GetHost(ctx), e.ID,
		))
	}

	// Check cancelation can be performed (either we're the last node, or the
	// node after us has already canceled the transaction, or the node after us
	// does not know about the transaction).
	if !e.Plan.CheckCanCancel(ctx, e.Client, e.Hop) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "cancellation_failed",
			"This transaction has not been cancelled by the next node on the "+
				"transaction plan: %s",
			e.Plan.Hops[e.Hop+1].Mint,
		))
	}

	// Cancel will idempotently cancel the transaction on all hops that are
	// involving this mint.
	err = e.Cancel(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "cancellation_failed",
			"The cancellation execution failed for the transaction: %s", e.ID,
		))
	}

	// We mark the transaction as cancelled if the hop of this is the minimal
	// one for this mint. Cancelation checks only use operations and crossings
	// so the status of a transaction is mostly indicative, but we want to mark
	// it as cancelled only after it is cancelled at all hops.
	if e.Hop == *minHop {
		e.Tx.Status = mint.TxStCanceled
		err = e.Tx.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	ops, err := model.LoadCanonicalOperationsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	crs, err := model.LoadCanonicalCrossingsByTransaction(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	// Commit the transaction as well as operations and crossings as canceled..
	db.Commit(ctx)

	err = e.Propagate(ctx)
	if err != nil {
		// If cancellation propagation failed we log it and trigger an
		// asyncrhonous one. In any case the node before us will check on us
		// before attempting to settle as well.
		mint.Logf(ctx,
			"Cancellation propagation failed: transaction=%s hop=%d error=%s",
			e.ID, e.Hop, err.Error())
		err = async.Queue(ctx,
			task.NewPropagateCancellation(ctx,
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

// Cancel cancels idempotently the underlying operations and crossings at the
// current hop.
func (e *CancelTransaction) Cancel(
	ctx context.Context,
) error {
	if int(e.Hop) >= len(e.Plan.Hops) {
		return errors.Trace(errors.Newf(
			"Hop (%d) is higher than the transaction plan length (%d)",
			e.Hop, len(e.Plan.Hops)))
	}

	h := e.Plan.Hops[e.Hop]
	mint.Logf(ctx,
		"Executing cancellation plan: transaction=%s hop=%d", e.ID, e.Hop)

	// Cancel the OpAction (should always be defined)
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

		if op.Status == mint.TxStCanceled {
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

			// Restore the source balance if applicable (that is if the op
			// source is not owner of the asset, in which case the asset was
			// issued on the fly).
			var srcBalance *model.Balance
			if asset.Owner != op.Source {
				srcBalance, err = model.LoadCanonicalBalanceByAssetHolder(ctx,
					op.Asset, op.Source)
				if err != nil {
					return errors.Trace(err)
				} else if srcBalance == nil {
					return errors.Trace(errors.Newf(
						"Source has no balance in %s: %s", op.Asset, op.Source))
				}
				(*big.Int)(&srcBalance.Value).Add(
					(*big.Int)(&srcBalance.Value), (*big.Int)(&op.Amount))

				// Checks if the srcBalance is positive and not overflown.
				b := (*big.Int)(&srcBalance.Value)
				if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
					b.Cmp(new(big.Int)) < 0 {
					return errors.Trace(errors.Newf(
						"Invalid resulting balance for %s: %s",
						srcBalance.Holder, b.String()))
				}

				err = srcBalance.Save(ctx)
				if err != nil {
					return errors.Trace(err)
				}

				err = async.Queue(ctx,
					task.NewPropagateBalance(ctx, time.Now(), srcBalance.ID()))
				if err != nil {
					return errors.Trace(err)
				}
			}

			op.Status = mint.TxStCanceled
			err = op.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}

			mint.Logf(ctx,
				"Canceled operation: id=%s[%s] created=%q propagation=%s "+
					"asset=%s source=%s destination=%s amount=%s "+
					"status=%s transaction=%s",
				op.Owner, op.Token, op.Created, op.Propagation, op.Asset,
				op.Source, op.Destination, (*big.Int)(&op.Amount).String(),
				op.Status, *op.Transaction)
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
			a := h.CrAction

			offer, err := model.LoadCanonicalOfferByID(ctx, *a.CrossingOffer)
			if err != nil {
				return errors.Trace(err)
			} else if offer == nil {
				return errors.Trace(errors.Newf(
					"Offer not found: %s", *a.CrossingOffer))
			}

			(*big.Int)(&offer.Remainder).Add(
				(*big.Int)(&offer.Remainder), (*big.Int)(&cr.Amount))

			// Checks if the remainder is positive and not overflown.
			b := (*big.Int)(&offer.Remainder)
			if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
				b.Cmp(new(big.Int)) < 0 {
				return errors.Trace(errors.Newf(
					"Invalid resulting remainder: %s", b.String()))
			}
			// Set the offer as active if the remainder is not 0 and the offer
			// is not closed.
			if offer.Status != mint.OfStClosed && b.Cmp(new(big.Int)) > 0 {
				offer.Status = mint.OfStActive
			}

			err = offer.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}

			err = async.Queue(ctx,
				task.NewPropagateOffer(ctx, time.Now(), offer.ID()))
			if err != nil {
				return errors.Trace(err)
			}

			cr.Status = mint.TxStCanceled
			err = cr.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}

			mint.Logf(ctx,
				"Canceled crossing: id=%s[%s] created=%q offer=%s amount=%s "+
					"status=%s transaction=%s",
				cr.Owner, cr.Token, cr.Created, cr.Offer,
				(*big.Int)(&cr.Amount).String(), cr.Status, cr.Transaction)
		}
	}

	return nil
}

// Propagate the transaction cancellation. Current hop cancellation is already
// performed.
func (e *CancelTransaction) Propagate(
	ctx context.Context,
) error {
	if int(e.Hop)-1 >= 0 {
		m := e.Plan.Hops[e.Hop-1].Mint

		mint.Logf(ctx,
			"Propagating cancellation: transaction=%s hop=%d mint=%s",
			e.ID, e.Hop, m)

		_, err := e.Client.CancelTransaction(ctx, e.ID, e.Hop-1, m)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
