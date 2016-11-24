// OWNER: stan

package endpoint

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"golang.org/x/sync/errgroup"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
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

// TxActionType is the type of the action of a TxPlan
type TxActionType string

const (
	// TxActTpOperation is the action type for an operation creation.
	TxActTpOperation TxActionType = "operation"
	// TxActTpCrossing is the action type for a crossing creation.
	TxActTpCrossing TxActionType = "crossing"
)

// TxAction represents the action on each mint that makes up a transaction
// plan.
type TxAction struct {
	Mint   string
	Owner  string
	Type   TxActionType
	Amount *big.Int

	CrossingOffer *string

	OperationAsset       *string
	OperationSource      *string
	OperationDestination *string
}

// TxPlan is the plan associated with the transaction. It is constructed by
// each mint involved in the transaction.
type TxPlan []TxAction

// CreateTransaction creates a new transaction.
type CreateTransaction struct {
	Client *mint.Client

	// Parameters
	Hop         int8                 // propagated
	ID          string               // propagated
	Token       string               // propagated
	Owner       string               // canonical, propagated
	Pair        []mint.AssetResource // canonical
	Amount      big.Int              // canonical
	Destination string               // canonical
	Path        []string             // canonical

	// State
	Tx     *model.Transaction
	Plan   TxPlan
	Offers []mint.OfferResource
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
		id, owner, token, err := ValidateID(ctx, r.PostFormValue("id"))
		if err != nil {
			return errors.Trace(err)
		}
		e.ID = *id
		e.Token = *token
		e.Owner = *owner

		hop, err := strconv.ParseInt(r.PostFormValue("hop"), 10, 8)
		if err != nil {
			return errors.Trace(errors.NewUserErrorf(err,
				400, "hop_invalid",
				"The transaction hop provided is invalid: %s. Transaction "+
					"hops must be 8bits integers.",
				r.PostFormValue("hop"),
			))
		}
		e.Hop = int8(hop)

		return nil
	}

	e.Owner = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		env.Get(ctx).Config[mint.EnvCfgMintHost])
	e.Hop = int8(0)

	// Validate asset pair.
	pair, err := ValidateAssetPair(ctx, r.PostFormValue("pair"))
	if err != nil {
		return errors.Trace(err) // 400
	}
	e.Pair = pair

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

	// Create canonical transaction locally.
	tx, err := model.CreateCanonicalTransaction(ctx,
		authentication.Get(ctx).User.Token,
		e.Owner,
		e.Pair[0].Name, e.Pair[1].Name,
		model.Amount(e.Amount),
		e.Destination, model.OfPath(e.Path),
		model.TxStReserved)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}
	e.Tx = tx
	e.ID = fmt.Sprintf("%s[%s]", tx.Owner, tx.Token)

	// Store the transcation in memory so that it's available through API
	// before we commit.
	txStore.Put(ctx, e.ID, tx)
	defer txStore.Clear(ctx, e.ID)

	err = e.ComputePlan(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The plan computation for the transaction failed: %s", e.ID,
		))
	}

	err = e.ExecutePlan(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "transaction_failed",
			"The plan execution for the transaction failed: %s", e.ID,
		))
	}

	err = e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "transaction_failed",
			"The transaction failed to propagate to all required mints: %s.",
			e.ID,
		))
	}

	// TODO(stan): reserve operation or crossing

	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{}, nil
}

// ExecutePropagated executes the creation of a propagated transaction
// (involved mint).
func (e *CreateTransaction) ExecutePropagated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	// TODO(stan): retrieve transaction from ID

	// Store the transcation in memory so that it's available through API
	// before we commit.
	//txStore.Put(ctx, e.ID, tx)
	//defer txStore.Clear(e.ID)

	err := e.ComputePlan(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "transaction_failed",
			"The plan computation for the transaction failed.",
		))
	}

	err = e.Propagate(ctx)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "transaction_failed",
			"The transaction failed to propagate to all required mints: %s.",
			e.ID,
		))
	}

	// TODO(stan): reserve operation or crossing

	return ptr.Int(http.StatusCreated), &svc.Resp{}, nil
}

// ComputePlan retrieves the offers of the path and compute the transaction
// plan.
func (e *CreateTransaction) ComputePlan(
	ctx context.Context,
) error {
	g, ctx := errgroup.WithContext(ctx)

	e.Offers = make([]mint.OfferResource, len(e.Tx.Path))

	for i, id := range e.Tx.Path {
		i, id := i, id
		g.Go(func() error {
			offer, err := e.Client.RetrieveOffer(ctx, id)
			if err != nil {
				return errors.Trace(err)
			}
			// TODO(stan): validate that the offer owner owns the base asset.
			e.Offers[i] = *offer
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return errors.Trace(err)
	}

	// As offers are asks (base asset offered in exchange for quote asset), the
	// transaction A/D requires offers:
	// B/A
	// C/B
	// D/C

	e.Plan = TxPlan{}

	// FIRST PASS: consists in computing the actions for all operations,
	// leaving the amounts empty.

	// Generate first action.
	_, host, err := mint.UsernameAndMintHostFromAddress(ctx, e.Owner)
	if err != nil {
		return errors.Trace(err)
	}
	e.Plan = append(e.Plan, TxAction{
		Mint:                 host,
		Owner:                e.Tx.Owner,
		Type:                 TxActTpOperation,
		OperationAsset:       &e.Tx.BaseAsset,
		Amount:               nil, // computed on second pass
		OperationDestination: nil, // computed by next offer
		OperationSource:      &e.Tx.Owner,
	})

	// Generate actions from path of offers.
	for i, offer := range e.Offers {
		offer := offer
		pair, err := mint.AssetResourcesFromPair(ctx, offer.Pair)
		if err != nil {
			return errors.Trace(err)
		}
		// Compare the previous operation asset with the offer quote asset.
		if pair[1].Name != *e.Plan[2*i].OperationAsset {
			return errors.Trace(errors.Newf(
				"Operation/Offer asset mismatch at offer %s: %s expected %s.",
				offer.ID, pair[0].Name, *e.Plan[2*i].OperationAsset))
		}
		// Fill the previous operation destination.
		e.Plan[2*i].OperationDestination = &offer.Owner
		// Add the crossing action.
		_, host, err := mint.UsernameAndMintHostFromAddress(ctx, offer.Owner)
		if err != nil {
			return errors.Trace(err)
		}
		e.Plan = append(e.Plan, TxAction{
			Mint:          host,
			Owner:         offer.Owner,
			Type:          TxActTpCrossing,
			CrossingOffer: &offer.ID,
			Amount:        nil, // computed on second pass
		})
		// Add the next operation action.
		_, host, err = mint.UsernameAndMintHostFromAddress(ctx, pair[0].Owner)
		if err != nil {
			return errors.Trace(err)
		}
		if offer.Owner != pair[0].Owner {
			return errors.Trace(errors.Newf(
				"Offer owner (%s) is not the offer base asset owner (%s).",
				offer.Owner, pair[0].Owner))
		}
		e.Plan = append(e.Plan, TxAction{
			Mint:                 host,
			Owner:                pair[0].Owner,
			Type:                 TxActTpOperation,
			OperationAsset:       &pair[0].Name,
			Amount:               nil,            // computed on second pass
			OperationDestination: nil,            // computed by next offer
			OperationSource:      &pair[0].Owner, // issuing operation
		})
	}
	// Compare the last operation asset to the transaction quote asset.
	if e.Tx.QuoteAsset != *e.Plan[len(e.Plan)-1].OperationAsset {
		return errors.Trace(errors.Newf(
			"Operation/Transaction asset mismatchs: %s expected %s.",
			e.Tx.QuoteAsset, *e.Plan[len(e.Plan)-1].OperationAsset))
	}
	// Compute the last operation destination.
	e.Plan[len(e.Plan)-1].OperationDestination = &e.Tx.Destination

	// SECOND PASS: consists in computing the amounts for each operations.

	// Compute the last amount, this is the transaction amount.
	e.Plan[len(e.Plan)-1].Amount = (*big.Int)(&e.Tx.Amount)

	// Compute amounts for each action.
	for i := len(e.Offers) - 1; i >= 0; i-- {
		// Offer amounts are expressed in quote asset
		basePrice, quotePrice, err := ValidatePrice(ctx, e.Offers[i].Price)
		if err != nil {
			return errors.Trace(err)
		}
		amount := new(big.Int).Mul(e.Plan[2*(i+1)].Amount, quotePrice)
		amount, remainder := new(big.Int).QuoRem(
			amount, basePrice, new(big.Int))

		// Transactions do cross offers on non congruent prices, costing one
		// base unit of quote asset. If the difference of scale between assets
		// is high, this can cost a lot to the owner of the transaction (but if
		// they issued it, they know).
		if remainder.Cmp(big.NewInt(0)) > 0 {
			amount = new(big.Int).Add(amount, big.NewInt(1))
		}

		e.Plan[2*i].Amount = amount
		e.Plan[2*i+1].Amount = amount
	}

	logLine := fmt.Sprintf("Transaction plan for %s:", e.ID)
	for i, a := range e.Plan {
		switch a.Type {
		case TxActTpOperation:
			logLine += fmt.Sprintf(
				"\n  [%d:%s] mint=%s amount=%s "+
					"asset=%s source=%s destination=%s ",
				i, a.Type, a.Mint, a.Amount.String(),
				*a.OperationAsset, *a.OperationSource, *a.OperationDestination)
		case TxActTpCrossing:
			logLine += fmt.Sprintf(
				"\n  [%d:%s] mint=%s amount=%s "+
					"offer=%s pair=%s price=%s",
				i, a.Type, a.Mint, a.Amount.String(),
				*a.CrossingOffer, e.Offers[i/2].Pair, e.Offers[i/2].Price)
		}
	}
	logging.Logf(ctx, logLine)

	return nil
}

// ExecutePlan executes the action locally at the current hop.
func (e *CreateTransaction) ExecutePlan(
	ctx context.Context,
) error {
	if int(e.Hop) >= len(e.Plan) {
		return errors.Trace(errors.Newf(
			"Hop (%d) is higher than the transaction plan length (%d)",
			e.Hop, len(e.Plan)))
	}

	owner := fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		env.Get(ctx).Config[mint.EnvCfgMintHost])

	a := e.Plan[e.Hop]
	if a.Owner != owner {
		return errors.Trace(errors.Newf(
			"Action owner mismatch at current hop (%d): %s expected %s.",
			e.Hop, a.Owner, owner))
	}

	switch a.Type {
	case TxActTpOperation:

		r, err := mint.AssetResourceFromName(ctx, *a.OperationAsset)
		if err != nil {
			return errors.Trace(err)
		}

		asset, err := model.LoadAssetByOwnerCodeScale(ctx,
			a.Owner, r.Code, r.Scale)
		if err != nil {
			return errors.Trace(err)
		} else if asset == nil {
			return errors.Trace(errors.Newf(
				"Asset not found: %s", *a.OperationAsset))
		}

		var srcBalance *model.Balance
		if a.OperationSource != nil && r.Owner != *a.OperationSource {
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
		if a.OperationDestination != nil && r.Owner != *a.OperationDestination {
			dstBalance, err = model.LoadOrCreateBalanceByAssetHolder(ctx,
				asset.User,
				asset.Owner,
				*a.OperationAsset, *a.OperationDestination)
			if err != nil {
				return errors.Trace(err)
			}
		}

		op, err := model.CreateCanonicalOperation(ctx,
			asset.User,
			asset.Owner,
			*a.OperationAsset,
			*a.OperationSource, *a.OperationDestination,
			model.Amount(*a.Amount),
			model.TxStReserved,
			ptr.Str(fmt.Sprintf("%s[%s]", e.Tx.Owner, e.Tx.Token)))
		if err != nil {
			return errors.Trace(err)
		}

		// Check the balances but only update the source balance. The
		// destination balance will get updated when the operation is confirmed
		// and the source balance will get reverted if it cancels.

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
		}

		if srcBalance != nil {
			(*big.Int)(&srcBalance.Value).Sub(
				(*big.Int)(&srcBalance.Value), (*big.Int)(&op.Amount))

			// Checks if the srcBalance is positive and not overflown.
			b := (*big.Int)(&srcBalance.Value)
			if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
				b.Cmp(new(big.Int)) < 0 {
				return errors.Trace(errors.Newf(
					"Invalid resulting balance: %s", b.String()))
			}
			err = srcBalance.Save(ctx)
			if err != nil {
				return errors.Trace(err)
			}
		}

		logging.Logf(ctx,
			"Reserved operation: user=%s id=%s[%s] created=%q propagation=%s "+
				"asset=%s source=%s destination=%s amount=%s "+
				"status=%s transaction=%s",
			op.User, op.Owner, op.Token, op.Created, op.Propagation,
			op.Asset, op.Source, op.Destination,
			(*big.Int)(&op.Amount).String(), op.Status, *op.Transaction)

	case TxActTpCrossing:

		owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx,
			*a.CrossingOffer)

		offer, err := model.LoadCanonicalOfferByOwnerToken(ctx,
			owner, token)
		if err != nil {
			return errors.Trace(err)
		} else if offer == nil {
			return errors.Trace(errors.Newf(
				"Offer not found: %s", *a.CrossingOffer))
		}

		cr, err := model.CreateCrossing(ctx,
			offer.User,
			offer.Owner,
			*a.CrossingOffer,
			model.Amount(*a.Amount),
			model.TxStReserved,
			fmt.Sprintf("%s[%s]", e.Tx.Owner, e.Tx.Token))
		if err != nil {
			return errors.Trace(err)
		}

		logging.Logf(ctx,
			"Reserved crossing: user=%s id=%s[%s] created=%q "+
				"offer=%s amount=%s status=%s transaction=%s",
			cr.User, cr.Owner, cr.Token, cr.Created,
			cr.Offer, (*big.Int)(&cr.Amount).String(),
			cr.Status, cr.Transaction)

		// TODO(stan) decrease offer remainder
	}

	return nil
}

// Propagate recursively propagates to the next mint in the chain of mint
// involved in a transaction.
func (e *CreateTransaction) Propagate(
	ctx context.Context,
) error {
	return nil
}
