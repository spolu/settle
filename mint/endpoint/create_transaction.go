// OWNER: stan

package endpoint

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"goji.io/pat"

	"golang.org/x/sync/errgroup"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/authentication"
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
	Plan *TxPlan
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

	if authentication.Get(ctx).Status != authentication.AutStSucceeded {
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

		return nil
	}

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

	return nil
}

// Execute executes the endpoint.
func (e *CreateTransaction) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	if authentication.Get(ctx).Status == authentication.AutStSucceeded {
		return e.ExecuteCanonical(ctx)
	}
	return e.ExecutePropagated(ctx)
}

// ExecuteCanonical executes the creation of a canonical transaction (owner
// mint).
func (e *CreateTransaction) ExecuteCanonical(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	// No need to lock the transaction here as we are the only mint to know its
	// ID before it propagagtes.
	txStore.Init(ctx, e.ID)

	// Create canonical transaction locally.
	tx, err := model.CreateCanonicalTransaction(ctx,
		e.Owner, e.BaseAsset, e.QuoteAsset, model.Amount(e.Amount),
		e.Destination, model.OfPath(e.Path), mint.TxStReserved)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}
	e.Tx = tx
	e.ID = fmt.Sprintf("%s[%s]", tx.Owner, tx.Token)

	// Store the transcation in memory so that it's available through API
	// before we commit.
	txStore.Store(ctx, e.ID, tx)
	defer txStore.Clear(ctx, e.ID)

	e.Plan = txStore.GetPlan(ctx, e.ID)
	if e.Plan == nil {
		plan, err := ComputePlan(ctx, e.Client, e.Tx)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "transaction_failed",
				"The plan computation for the transaction failed: %s", e.ID,
			))
		}
		txStore.StorePlan(ctx, e.ID, plan)
		e.Plan = plan
	}

	err = e.ExecutePlan(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The plan execution failed at hop %d for transaction: %s",
			e.Hop, e.ID,
		))
	}

	err = e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The transaction failed to propagate to required mints: %s.",
			e.ID,
		))
	}

	// Check all the hops in the transaction in parallel before committing (as
	// we are the canonical mint for it).
	g, ctx := errgroup.WithContext(ctx)

	for hop := 1; hop < len(e.Plan.Actions); hop++ {
		hop := hop
		g.Go(func() error {
			err = e.Plan.Check(ctx, e.Client, int8(hop))
			if err != nil {
				return errors.Trace(err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"Failed to check plan for transaction %s",
			e.ID,
		))
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			txStore.Get(ctx, e.ID),
			txStore.GetOperations(ctx, e.ID),
			txStore.GetCrossings(ctx, e.ID),
		)),
	}, nil
}

// ExecutePropagated executes the creation of a propagated transaction
// (involved mint).
func (e *CreateTransaction) ExecutePropagated(
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
		// underlying db.Transaction so that the whole transaction is run on
		// one single underlying db.Transaction (more consistent and avoids
		// locking issues).
		dbTx := txStore.GetDBTransaction(ctx, e.ID)
		ctx = db.WithTransaction(ctx, *dbTx)
		mint.Logf(ctx, "Transaction: reuse %s", dbTx.Token)

		e.Tx = txStore.Get(ctx, e.ID)
		e.Owner = e.Tx.Owner
		e.BaseAsset = e.Tx.BaseAsset
		e.QuoteAsset = e.Tx.QuoteAsset
		e.Amount = big.Int(e.Tx.Amount)
		e.Destination = e.Tx.Destination
		e.Path = []string(e.Tx.Path)
	} else {
		mustCommit = true
		ctx = db.Begin(ctx)
		defer db.LoggedRollback(ctx)

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
			e.BaseAsset, e.QuoteAsset,
			model.Amount(e.Amount),
			e.Destination, model.OfPath(e.Path),
			mint.TxStReserved, transaction.Lock)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
		e.Tx = tx

		// Store the transcation in memory so that it's available through API
		// before we commit.
		txStore.Store(ctx, e.ID, tx)
		defer txStore.Clear(ctx, e.ID)
	}

	e.Plan = txStore.GetPlan(ctx, e.ID)
	if e.Plan == nil {
		plan, err := ComputePlan(ctx, e.Client, e.Tx)
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				402, "transaction_failed",
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

	// Check the plan at previous hop before we execute this hop, to convince
	// ourselves that the funds are reserved!
	err := e.Plan.Check(ctx, e.Client, e.Hop-1)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"Failed to check plan at hop %d for transaction %s",
			e.Hop-1, e.ID,
		))
	}

	err = e.ExecutePlan(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The plan execution failed at hop %d for transaction: %s",
			e.Hop, e.ID,
		))
	}

	// We unlock the tranaction before propagating.
	unlock()

	err = e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The transaction failed to propagate to required mints: %s.",
			e.ID,
		))
	}

	if mustCommit {
		db.Commit(ctx)
	}

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			txStore.Get(ctx, e.ID),
			txStore.GetOperations(ctx, e.ID),
			txStore.GetCrossings(ctx, e.ID),
		)),
	}, nil
}

// Propagate recursively propagates to the next mint in the chain of mint
// involved in a transaction.
func (e *CreateTransaction) Propagate(
	ctx context.Context,
) error {
	if int(e.Hop)+1 < len(e.Plan.Actions) {

		m := e.Plan.Actions[e.Hop+1].Mint

		mint.Logf(ctx,
			"Propagating transaction: transaction=%s hop=%d mint=%s",
			e.ID, e.Hop, m)

		_, err := e.Client.PropagateTransaction(ctx, e.ID, e.Hop+1, m)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// ExecutePlan executes the action locally at the current hop.
func (e *CreateTransaction) ExecutePlan(
	ctx context.Context,
) error {
	if int(e.Hop) >= len(e.Plan.Actions) {
		return errors.Trace(errors.Newf(
			"Hop (%d) is higher than the transaction plan length (%d)",
			e.Hop, len(e.Plan.Actions)))
	}

	a := e.Plan.Actions[e.Hop]
	if a.IsExecuted {
		mint.Logf(ctx,
			"Skipping transaction plan: transaction=%s hop=%d", e.ID, e.Hop)
		return nil
	}
	mint.Logf(ctx,
		"Executing transcation plan: transaction=%s hop=%d", e.ID, e.Hop)

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

		var srcBalance *model.Balance
		if a.OperationSource != nil && asset.Owner != *a.OperationSource {
			srcBalance, err = model.LoadBalanceByAssetHolder(ctx,
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
			dstBalance, err = model.LoadOrCreateBalanceByAssetHolder(ctx,
				asset.Owner, *a.OperationAsset, *a.OperationDestination)
			if err != nil {
				return errors.Trace(err)
			}
		}

		op, err := model.CreateCanonicalOperation(ctx,
			asset.Owner, *a.OperationAsset,
			*a.OperationSource, *a.OperationDestination,
			model.Amount(*a.Amount), mint.TxStReserved, &e.ID, &e.Hop)
		if err != nil {
			return errors.Trace(err)
		}

		// Store the newly created operation so that it's available when the
		// transaction is returned from this mint.
		txStore.StoreOperation(ctx, e.ID, op)

		// Check the balances but only update the source balance. The
		// destination balance will get updated when the operation is settled
		// and the source balance will get reverted if it cancels.

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
		}

		mint.Logf(ctx,
			"Reserved operation: id=%s[%s] created=%q propagation=%s "+
				"asset=%s source=%s destination=%s amount=%s "+
				"status=%s transaction=%s",
			op.Owner, op.Token, op.Created, op.Propagation, op.Asset,
			op.Source, op.Destination, (*big.Int)(&op.Amount).String(),
			op.Status, *op.Transaction)

	case TxActTpCrossing:

		offer, err := model.LoadCanonicalOfferByID(ctx,
			*a.CrossingOffer)
		if err != nil {
			return errors.Trace(err)
		} else if offer == nil {
			return errors.Trace(errors.Newf(
				"Offer not found: %s", *a.CrossingOffer))
		}

		cr, err := model.CreateCrossing(ctx,
			offer.Owner, *a.CrossingOffer, model.Amount(*a.Amount),
			mint.TxStReserved, e.ID, e.Hop)
		if err != nil {
			return errors.Trace(err)
		}

		// Store the newly created crossing so that it's available when the
		// transaction is returned from this mint.
		txStore.StoreCrossing(ctx, e.ID, cr)

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

		mint.Logf(ctx,
			"Reserved crossing: id=%s[%s] created=%q offer=%s amount=%s "+
				"status=%s transaction=%s",
			cr.Owner, cr.Token, cr.Created, cr.Offer,
			(*big.Int)(&cr.Amount).String(), cr.Status, cr.Transaction)
	}

	return nil
}
