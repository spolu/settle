package endpoint

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"goji.io/pat"

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
)

const (
	// EndPtCreateTransaction creates a new transaction.
	EndPtCreateTransaction EndPtName = "CreateTransaction"
)

func init() {
	registrar[EndPtCreateTransaction] = NewCreateTransaction
}

// CreateTransaction creates a new transaction.
type CreateTransaction struct {
	Client *mint.Client

	// Parameters
	Hop         int8
	ID          string
	Owner       string
	BaseAsset   string
	QuoteAsset  string
	Amount      big.Int
	Destination string
	Path        []string

	// State
	Tx   *model.Transaction
	Plan *plan.TxPlan
}

// NewCreateTransaction constructs and initialiezes the endpoint.
func NewCreateTransaction(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}
	return &CreateTransaction{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *CreateTransaction) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	switch authentication.Get(ctx).Status {
	case authentication.AutStSkipped:
		// Validate id.
		id, owner, _, err := ValidateID(ctx, pat.Param(r, "transaction"))
		if err != nil {
			return errors.Trace(err)
		}
		e.ID = *id
		e.Owner = *owner

		// Validate hop.
		hop, err := ValidateHop(ctx, r.PostFormValue("hop"))
		if err != nil {
			return errors.Trace(err)
		}
		e.Hop = *hop

	case authentication.AutStSucceeded:
		e.Owner = fmt.Sprintf("%s@%s",
			authentication.Get(ctx).User.Username,
			mint.GetHost(ctx))
		e.Hop = int8(0)

		// Validate asset pair.
		pair, err := ValidateAssetPair(ctx, r.PostFormValue("pair"))
		if err != nil {
			return errors.Trace(err) // 400
		}
		e.BaseAsset = pair[0].Name
		e.QuoteAsset = pair[1].Name

		// Validate amount.
		amount, err := ValidateAmount(ctx, r.PostFormValue("amount"))
		if err != nil {
			return errors.Trace(err)
		}
		e.Amount = *amount

		// Validate destination.
		dstAddress, err := mint.NormalizedAddress(ctx, r.PostFormValue("destination"))
		if err != nil {
			return errors.Trace(errors.NewUserErrorf(err,
				400, "destination_invalid",
				"The destination address you provided is invalid: %s.",
				dstAddress,
			))
		}
		e.Destination = dstAddress

		// Validate path.
		if r.PostForm == nil {
			err := r.ParseMultipartForm(defaultMaxMemory)
			if err != nil {
				return errors.Trace(err) // 500
			}
		}
		path, err := ValidatePath(ctx, r.PostForm["path[]"])
		if err != nil {
			return errors.Trace(err)
		}
		e.Path = path
	}

	return nil
}

// Execute executes the endpoint.
func (e *CreateTransaction) Execute(
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

// ExecuteCanonical executes the creation of a canonical transaction (owner
// mint).
func (e *CreateTransaction) ExecuteCanonical(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	oCtx := ctx

	ctx = db.Begin(oCtx, "mint")
	defer db.LoggedRollback(ctx)

	// Create canonical transaction locally.
	tx, err := model.CreateCanonicalTransaction(ctx,
		e.Owner,
		e.BaseAsset,
		e.QuoteAsset,
		model.Amount(e.Amount),
		e.Destination,
		model.OfPath(e.Path),
		mint.TxStPending,
	)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}
	e.Tx = tx
	e.ID = e.Tx.ID()

	pl, err := plan.Compute(ctx, e.Client, e.Tx, false)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}
	e.Plan = pl

	// Commit the transaction in pending state.
	db.Commit(ctx)

	// At the canonical mint the propagation starts from a virtual Hop which is
	// the length of the plan hops plus one.
	e.Hop = int8(len(e.Plan.Hops))

	txn, err := e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The transaction failed to propagate to required mints: %s.",
			e.ID,
		))
	}

	// We just need to check that the resource we received from the last hop
	// matches our transaction plan.
	err = e.Plan.Check(ctx, txn, int8(e.Hop-1))
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"Failed to check plan for transaction %s",
			e.ID,
		))
	}

	ctx = db.Begin(oCtx, "mint")
	defer db.LoggedRollback(ctx)

	// Reload the transaction post propagation.
	tx, err = model.LoadTransactionByID(ctx, e.ID)
	if err != nil || tx == nil {
		return nil, nil, errors.Trace(err) // 500
	}
	e.Tx = tx

	switch e.Tx.Status {
	case mint.TxStPending:
		// Mark the transaction as reserved.
		e.Tx.Status = mint.TxStReserved
	case mint.TxStReserved:
		// No-op as the transaction was already marked as reserved during
		// propagation.
	default:
		return nil, nil, errors.Newf(
			"Unexpected transaction status %s: %s", e.Tx.Status, e.ID) // 500
	}

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

	// Commit the transaction in reserved state.
	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			tx, ops, crs,
		)),
	}, nil
}

// ExecutePropagated executes the creation of a propagated transaction
// (involved mint).
func (e *CreateTransaction) ExecutePropagated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	oCtx := ctx

	ctx = db.Begin(oCtx, "mint")
	defer db.LoggedRollback(ctx)

	tx, err := model.LoadTransactionByID(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}
	if tx != nil {
		e.Tx = tx
		e.Owner = e.Tx.Owner
		e.BaseAsset = e.Tx.BaseAsset
		e.QuoteAsset = e.Tx.QuoteAsset
		e.Amount = big.Int(e.Tx.Amount)
		e.Destination = e.Tx.Destination
		e.Path = []string(e.Tx.Path)
	} else {
		transaction, err := e.Client.RetrieveTransaction(ctx, e.ID, nil)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "transaction_failed",
				"Failed to retrieve transaction: %s", e.ID,
			))
		}

		owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, e.ID)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "transaction_failed",
				"Failed to retrieve transaction: %s", e.ID,
			))
		}
		e.Owner = owner
		p, err := mint.AssetResourcesFromPair(ctx, transaction.Pair)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "transaction_failed",
				"Failed to retrieve transaction: %s", e.ID,
			))
		}
		e.BaseAsset = p[0].Name
		e.QuoteAsset = p[1].Name
		e.Amount = *transaction.Amount
		e.Destination = transaction.Destination
		e.Path = transaction.Path

		// Create propagated transaction locally.
		tx, err := model.CreatePropagatedTransaction(ctx,
			token,
			time.Unix(0, transaction.Created*mint.TimeResolutionNs),
			e.Owner,
			e.BaseAsset,
			e.QuoteAsset,
			model.Amount(e.Amount),
			e.Destination,
			model.OfPath(e.Path),
			mint.TxStPending,
			transaction.Lock,
		)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
		e.Tx = tx
	}

	pl, err := plan.Compute(ctx, e.Client, e.Tx, false)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}
	e.Plan = pl

	if int(e.Hop) >= len(e.Plan.Hops) ||
		e.Plan.Hops[e.Hop].Mint != mint.GetHost(ctx) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "transaction_failed",
			"The hop provided (%d) does not match the current mint (%s) for "+
				"transaction: %s", e.Hop, mint.GetHost(ctx), e.ID,
		))
	}

	// Commit the transaction as pending if it was created.
	db.Commit(ctx)

	txn, err := e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The transaction failed to propagate to required mints: %s.",
			e.ID,
		))
	}

	// Check the plan of the txn received from previous hop (unles we're the
	// mint at hop 0) before we execute this hop, to convince ourselves that
	// the funds are reserved!
	if txn != nil {
		err = e.Plan.Check(ctx, txn, e.Hop-1)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "transaction_failed",
				"Failed to check plan at hop %d for transaction %s",
				e.Hop-1, e.ID,
			))
		}
	}

	ctx = db.Begin(oCtx, "mint")
	defer db.LoggedRollback(ctx)

	// Reload the transaction post propagation.
	tx, err = model.LoadTransactionByID(ctx, e.ID)
	if err != nil || tx == nil {
		return nil, nil, errors.Trace(err) // 500
	}
	e.Tx = tx

	// Idempotently execute plan for the transaction.
	err = e.ExecutePlan(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The plan execution failed at hop %d for transaction: %s",
			e.Hop, e.ID,
		))
	}

	switch e.Tx.Status {
	case mint.TxStPending:
		// Mark the transaction as reserved.
		e.Tx.Status = mint.TxStReserved
	case mint.TxStReserved:
		// No-op as the transaction was already marked as reserved during
		// propagation.
	default:
		return nil, nil, errors.Newf(
			"Unexpected transaction status %s: %s", e.Tx.Status, e.ID) // 500
	}

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

	// Commit the plan execution as well as the transaction status change.
	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			tx, ops, crs,
		)),
	}, nil
}

// ExecutePlan executes the Hop locally, performing the operation and the
// crossing action if applicable. ExecutePlan is executed in the context of a
// DB transaction and is idempotent (attempts to retrieve operations for that
// hop and transaction before executing them).
func (e *CreateTransaction) ExecutePlan(
	ctx context.Context,
) error {
	if int(e.Hop) >= len(e.Plan.Hops) {
		return errors.Trace(errors.Newf(
			"Hop (%d) is higher than the transaction plan length (%d)",
			e.Hop, len(e.Plan.Hops)))
	}

	h := e.Plan.Hops[e.Hop]
	mint.Logf(ctx,
		"Executing transaction plan: transaction=%s hop=%d", e.ID, e.Hop)

	// Execute the OpAction (should always be defined)
	if h.OpAction != nil {
		op, err := model.LoadCanonicalOperationByTransactionHop(ctx,
			e.ID, e.Hop)
		if err != nil {
			return errors.Trace(err)
		}
		if op != nil {
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

			var srcBalance *model.Balance
			if a.OperationSource != nil && asset.Owner != *a.OperationSource {
				srcBalance, err = model.LoadCanonicalBalanceByAssetHolder(ctx,
					*a.OperationAsset, *a.OperationSource)
				if err != nil {
					return errors.Trace(err)
				} else if srcBalance == nil {
					return errors.Trace(errors.Newf(
						"Source has no balance in %s: %s",
						*a.OperationAsset, *a.OperationSource))
				}
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

			op, err := model.CreateCanonicalOperation(ctx,
				a.Owner,
				*a.OperationAsset,
				*a.OperationSource,
				*a.OperationDestination,
				model.Amount(*a.Amount),
				mint.TxStReserved,
				&e.ID,
				&e.Hop,
			)
			if err != nil {
				return errors.Trace(err)
			}

			// Check the balances but only update the source balance. The
			// destination balance will get updated when the operation is
			// settled and the source balance will get reverted if it cancels.

			if dstBalance != nil {
				(*big.Int)(&dstBalance.Value).Add(
					(*big.Int)(&dstBalance.Value), (*big.Int)(&op.Amount))
				// Checks if the dstBalance is positive and not overflown.
				b := (*big.Int)(&dstBalance.Value)
				if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
					b.Cmp(new(big.Int)) < 0 {
					return errors.Trace(errors.Newf(
						"Invalid resulting balance for %s: %s",
						dstBalance.Holder, b.String()))
				}
			}

			// The srcBalance is not nil only if the transaction baseAsset is
			// not owned by the transaction owner (paying with a balance at
			// another mint). In which case we substract the balance. This is
			// quite dangerous as if a mint along the offer path fails its
			// reservation (on the way back), the funds will end-up reserved
			// and locked, as this is the only way to guarantee the necessary
			// commitment for the reservation.
			if srcBalance != nil {
				(*big.Int)(&srcBalance.Value).Sub(
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

			mint.Logf(ctx,
				"Reserved operation: id=%s[%s] created=%q propagation=%s "+
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
		}
		if cr != nil {
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

			if offer.Status != mint.OfStActive {
				return errors.Trace(errors.Newf(
					"Offer is not active (%s)", offer.Status))
			}

			cr, err := model.CreateCanonicalCrossing(ctx,
				a.Owner,
				*a.CrossingOffer,
				model.Amount(*a.Amount),
				mint.TxStReserved,
				e.ID,
				e.Hop,
			)
			if err != nil {
				return errors.Trace(err)
			}

			(*big.Int)(&offer.Remainder).Sub(
				(*big.Int)(&offer.Remainder), (*big.Int)(&cr.Amount))

			// Checks if the remainder is positive and not overflown.
			b := (*big.Int)(&offer.Remainder)
			if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
				b.Cmp(new(big.Int)) < 0 {
				return errors.Trace(errors.Newf(
					"Invalid resulting remainder: %s", b.String()))
			}
			// Set the offer as consumed if all funds are reserved. If the
			// transaction gets canceled, it'll get reverted.
			if b.Cmp(new(big.Int)) == 0 {
				offer.Status = mint.OfStConsumed
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

			mint.Logf(ctx,
				"Reserved crossing: id=%s[%s] created=%q offer=%s amount=%s "+
					"status=%s transaction=%s",
				cr.Owner, cr.Token, cr.Created, cr.Offer,
				(*big.Int)(&cr.Amount).String(), cr.Status, cr.Transaction)
		}
	}

	return nil
}

// Propagate recursively propagates to the next mint in the chain of mint
// involved in a transaction.
func (e *CreateTransaction) Propagate(
	ctx context.Context,
) (*mint.TransactionResource, error) {
	if int(e.Hop)-1 >= 0 {
		m := e.Plan.Hops[e.Hop-1].Mint

		mint.Logf(ctx,
			"Propagating transaction: transaction=%s hop=%d mint=%s",
			e.ID, e.Hop-1, m)

		txn, err := e.Client.PropagateTransaction(ctx, e.ID, e.Hop-1, m)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return txn, nil
	}
	return nil, nil
}
