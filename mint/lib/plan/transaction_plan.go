package plan

import (
	"context"
	"fmt"
	"math/big"
	"regexp"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
	"golang.org/x/sync/errgroup"
)

// TxActionType is the type of the action of a TxPlan
type TxActionType string

const (
	// TxActTpOperation is the action type for an operation creation.
	TxActTpOperation TxActionType = "operation"
	// TxActTpCrossing is the action type for a crossing creation.
	TxActTpCrossing TxActionType = "crossing"
)

// TxAction represents an action to be performed by a mint. Either an operation
// or crossing creation.
type TxAction struct {
	Owner  string
	Type   TxActionType
	Amount *big.Int

	CrossingOffer *string

	OperationAsset       *string
	OperationSource      *string
	OperationDestination *string
}

// TxHop is a list of action to be performed by the mint at the associated hop.
// hop 0 is the mint creating the transaction, hop (i) is the mint on the offer
// path at index (i-1).
type TxHop struct {
	Mint string

	OpAction *TxAction
	CrAction *TxAction
}

// TxPlan is the plan associated with the transaction. It is constructed by
// each mint involved in the transaction. Each hop represents a mint along the
// offer path starting with the mint initiating the transaction.
type TxPlan struct {
	Hops        []*TxHop
	Transaction string
}

// Compute retrieves the offers of the path and compute the transaction plan.
func Compute(
	ctx context.Context,
	client *mint.Client,
	tx *model.Transaction,
	shallow bool,
) (*TxPlan, error) {
	g, ctx := errgroup.WithContext(ctx)
	offers := make([]mint.OfferResource, len(tx.Path))

	for i, id := range tx.Path {
		i, id := i, id
		g.Go(func() error {
			if !shallow {
				offer, err := client.RetrieveOffer(ctx, id)
				if err != nil {
					return errors.Trace(err)
				}

				// Validate that the offer owner owns the base asset (enforced by
				// offer creation but good defense in depth to validate here).
				pair, err := mint.AssetResourcesFromPair(ctx, offer.Pair)
				if offer.Owner != pair[0].Owner {
					return errors.Newf(
						"Offer/BaseAsset owner mismatch at offer %s: %s expected "+
							"%s.", offer.ID, pair[0].Owner, offer.Owner)
				}

				offers[i] = *offer
			} else {
				// If we computing a shallow transaction plan, just store
				// minimal representation of offers.
				owner, id, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
				if err != nil {
					return errors.Trace(err)
				}
				offers[i] = mint.OfferResource{
					ID:    id,
					Owner: owner,
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	// As offers are asks (base asset offered in exchange for quote asset), the
	// transaction A/D requires offers:
	// B/A
	// C/B
	// D/C

	plan := TxPlan{
		Hops:        []*TxHop{},
		Transaction: tx.ID(),
	}

	// FIRST PASS: consists in computing the actions for all hops, leaving the
	// amounts empty.

	bAsset, err := mint.AssetResourceFromName(ctx, tx.BaseAsset)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// If this is a transaction whose baseAsset owner is not the transaction
	// owner, we inject a first hop with no action (for proper cancelation
	// propagation).
	offset := 0
	if bAsset.Owner != tx.Owner {
		_, host, err := mint.UsernameAndMintHostFromAddress(ctx, tx.Owner)
		if err != nil {
			return nil, errors.Trace(err)
		}
		offset = 1
		plan.Hops = append(plan.Hops, &TxHop{
			Mint: host,
		})
	}

	// Generate first hop which has one operation on the mint of the base
	// asset.
	_, host, err := mint.UsernameAndMintHostFromAddress(ctx, bAsset.Owner)
	if err != nil {
		return nil, errors.Trace(err)
	}

	h := TxHop{
		Mint: host,
	}
	if !shallow {
		h.OpAction = &TxAction{
			Owner:                bAsset.Owner,
			Type:                 TxActTpOperation,
			OperationAsset:       &bAsset.Name,
			Amount:               nil, // computed on second pass
			OperationDestination: nil, // computed by next offer
			OperationSource:      &tx.Owner,
		}
	}
	plan.Hops = append(plan.Hops, &h)

	// Generate actions from path of offers.
	for i, offer := range offers {
		hop := i + 1 + offset
		offer := offer
		_, host, err := mint.UsernameAndMintHostFromAddress(ctx, offer.Owner)
		if err != nil {
			return nil, errors.Trace(err)
		}

		if !shallow {
			pair, err := mint.AssetResourcesFromPair(ctx, offer.Pair)
			if err != nil {
				return nil, errors.Trace(err)
			}
			// Compare the previous operation asset with the offer quote asset.
			if pair[1].Name != *plan.Hops[hop-1].OpAction.OperationAsset {
				return nil, errors.Trace(errors.Newf(
					"Operation/Offer asset mismatch at offer %s: %s expected %s.",
					offer.ID, pair[1].Name,
					*plan.Hops[hop-1].OpAction.OperationAsset))
			}
			// Fill the previous operation destination.
			plan.Hops[hop-1].OpAction.OperationDestination = &offer.Owner

			// Compute the hop for the current offer
			if offer.Owner != pair[0].Owner {
				return nil, errors.Trace(errors.Newf(
					"Offer owner (%s) is not the offer base asset owner (%s).",
					offer.Owner, pair[0].Owner))
			}
			plan.Hops = append(plan.Hops, &TxHop{
				Mint: host,
				CrAction: &TxAction{
					Owner:         offer.Owner,
					Type:          TxActTpCrossing,
					CrossingOffer: &offer.ID,
					Amount:        nil, // computed on second pass
				},
				OpAction: &TxAction{
					Owner:                pair[0].Owner,
					Type:                 TxActTpOperation,
					OperationAsset:       &pair[0].Name,
					Amount:               nil,            // computed on second pass
					OperationDestination: nil,            // computed by next offer
					OperationSource:      &pair[0].Owner, // issuing operation
				},
			})
		} else {
			plan.Hops = append(plan.Hops, &TxHop{
				Mint: host,
			})
		}
	}

	// If we're computing a shallow plan, we're good to return.
	if shallow {
		return &plan, nil
	}

	// Compare the last operation asset to the transaction quote asset.
	if tx.QuoteAsset != *plan.Hops[len(plan.Hops)-1].OpAction.OperationAsset {
		return nil, errors.Trace(errors.Newf(
			"Operation/Transaction asset mismatchs: %s expected %s.",
			tx.QuoteAsset,
			*plan.Hops[len(plan.Hops)-1].OpAction.OperationAsset))
	}
	// Compute the last operation destination.
	plan.Hops[len(plan.Hops)-1].OpAction.OperationDestination = &tx.Destination

	// SECOND PASS: consists in computing the amounts for each operations.

	// Compute the amount of the last hop operationw as the transaction amount.
	plan.Hops[len(plan.Hops)-1].OpAction.Amount = (*big.Int)(&tx.Amount)

	// Compute amounts for each action.
	for i := len(offers) - 1; i >= 0; i-- {
		hop := i + 1 + offset
		// Offer amounts are expressed in quote asset
		basePrice, quotePrice, err := ExtractPrice(ctx, offers[i].Price)
		if err != nil {
			return nil, errors.Trace(err)
		}
		amount := new(big.Int).Mul(
			plan.Hops[hop].OpAction.Amount,
			quotePrice)
		amount, remainder := new(big.Int).QuoRem(
			amount, basePrice, new(big.Int))

		// Transactions do cross offers on non congruent prices, costing one
		// base unit of quote asset. If the difference of scale between assets
		// is high, this can cost a lot to the owner of the transaction (but if
		// they issued it, they know).
		if remainder.Cmp(big.NewInt(0)) > 0 {
			amount = new(big.Int).Add(amount, big.NewInt(1))
		}

		plan.Hops[hop].CrAction.Amount = amount
		plan.Hops[hop-1].OpAction.Amount = amount
	}

	logLine := fmt.Sprintf("Transaction plan for %s:", plan.Transaction)
	for i, h := range plan.Hops {
		logLine += fmt.Sprintf("\n  [%d] mint=%s", i, h.Mint)
		if h.OpAction != nil {
			a := h.OpAction
			logLine += fmt.Sprintf("\n    [%s] amount=%s asset=%s "+
				"source=%s destination=%s ",
				a.Type, a.Amount.String(), *a.OperationAsset,
				*a.OperationSource, *a.OperationDestination)
		}
		if h.CrAction != nil {
			a := h.CrAction
			logLine += fmt.Sprintf(
				"\n    [%s ] amount=%s offer=%s pair=%s price=%s",
				a.Type, a.Amount.String(), *a.CrossingOffer,
				offers[i-offset-1].Pair, offers[i-offset-1].Price)
		}
	}
	mint.Logf(ctx, logLine)

	return &plan, nil
}

// Check checks that the plan was properly executed at the specified hop by
// retrieving the transaction ont that mint and checking the actions against
// the advertised operations and crossings.
func (p *TxPlan) Check(
	ctx context.Context,
	transaction *mint.TransactionResource,
	hop int8,
) error {
	h := p.Hops[hop]

	if h.OpAction != nil {
		a := h.OpAction

		operation := (*mint.OperationResource)(nil)
		for _, op := range transaction.Operations {
			op := op
			if op.TransactionHop != nil && *op.TransactionHop == hop {
				operation = &op
			}
		}
		if operation == nil {
			return errors.Newf("Operation at hop %d not found", hop)
		}
		if operation.Owner != a.Owner {
			return errors.Newf("Operation at hop %d owner mismatch: "+
				"%s expected %s",
				hop, operation.Owner, a.Owner)
		}
		if operation.Amount.Cmp(a.Amount) != 0 {
			return errors.Newf("Operation at hop %d amount mismatch: "+
				"%s expected %s",
				hop, operation.Amount.String(), a.Amount.String())
		}
		if operation.Asset != *a.OperationAsset {
			return errors.Newf("Operation at hop %d asset mismatch: "+
				"%s expected %s",
				hop, operation.Asset, *a.OperationAsset)
		}
		if operation.Source != *a.OperationSource {
			return errors.Newf("Operation at hop %d source mismatch: "+
				"%s expected %s",
				hop, operation.Source, *a.OperationSource)
		}
		if operation.Destination != *a.OperationDestination {
			return errors.Newf("Operation at hop %d destination mismatch: "+
				"%s expected %s",
				hop, operation.Destination, *a.OperationDestination)
		}
	}

	if h.CrAction != nil {
		a := h.CrAction

		crossing := (*mint.CrossingResource)(nil)
		for _, cr := range transaction.Crossings {
			cr := cr
			if cr.TransactionHop == hop {
				crossing = &cr
			}
		}
		if crossing == nil {
			return errors.Newf("Crossing at hop %d not found", hop)
		}
		if crossing.Owner != a.Owner {
			return errors.Newf("Crossing at hop %d owner mismatch: "+
				"%s expected %s",
				hop, crossing.Owner, a.Owner)
		}
		if crossing.Amount.Cmp(a.Amount) != 0 {
			return errors.Newf("Crossing at hop %d amount mismatch: "+
				"%s expected %s",
				hop, crossing.Amount.String(), a.Amount.String())
		}
		if crossing.Offer != *a.CrossingOffer {
			return errors.Newf("Crossing at hop %d offer mismatch: "+
				"%s expected %s",
				hop, crossing.Offer, *a.CrossingOffer)
		}
	}

	return nil
}

// MinMaxHop returns the lowest and highest hops for the local mint in the
// transaction plan. Works on a shallow plan.
func (p *TxPlan) MinMaxHop(
	ctx context.Context,
) (*int8, *int8, error) {
	min := int8(-1)
	max := int8(-1)
	for i, h := range p.Hops {
		if h.Mint == mint.GetHost(ctx) {
			max = int8(i)
			if min == -1 {
				min = int8(i)
			}
		}
	}
	if max == -1 {
		return nil, nil, errors.Newf(
			"This mint is not part of the transction plan.")
	}
	return &min, &max, nil
}

// CheckShouldCancel checks whether the next node on the offer path has
// canceled. If so, no need to settle, we can cancel instead. Works on a
// shallow plan.
func (p *TxPlan) CheckShouldCancel(
	ctx context.Context,
	client *mint.Client,
	hop int8,
) bool {
	// If we're the recipient we should not cancel unless instructed.
	if hop == int8(len(p.Hops)-1) {
		return false
	}

	txn, err := client.RetrieveTransaction(ctx,
		p.Transaction, &p.Hops[hop+1].Mint)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case mint.ErrMintClient:
			// If we get a legit 404 transaction_not_found from the mint, it
			// indicates that the transaction never propagated there or the
			// mint failed to persist it, so it's safe to cancel.
			if err.ErrCode == "transaction_not_found" {
				return true
			}
		default:
			return false
		}
	}

	operation := (*mint.OperationResource)(nil)
	for _, op := range txn.Operations {
		op := op
		if op.TransactionHop != nil && *op.TransactionHop == hop+1 {
			operation = &op
		}
	}
	if operation != nil && operation.Status == mint.TxStCanceled {
		return true
	}

	return false
}

// CheckCanCancel checks that this node is authorized to cancel the transaction
// (this is the node with higher hop or the node above it has already canceled
// the transaction). Works on a shallow plan.
func (p *TxPlan) CheckCanCancel(
	ctx context.Context,
	client *mint.Client,
	hop int8,
) bool {
	if hop == int8(len(p.Hops)-1) {
		return true
	}

	txn, err := client.RetrieveTransaction(ctx,
		p.Transaction, &p.Hops[hop+1].Mint)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case mint.ErrMintClient:
			// If we get a legit 404 transaction_not_found from the mint, it
			// indicates that the transaction never propagated there or the
			// mint failed to persist it, so it's safe to cancel.
			if err.ErrCode == "transaction_not_found" {
				return true
			}
		default:
			return false
		}
	}

	operation := (*mint.OperationResource)(nil)
	for _, op := range txn.Operations {
		op := op
		if op.TransactionHop != nil && *op.TransactionHop == hop+1 {
			operation = &op
		}
	}
	if operation != nil && operation.Status != mint.TxStCanceled {
		return false
	}

	crossing := (*mint.CrossingResource)(nil)
	for _, cr := range txn.Crossings {
		cr := cr
		if cr.TransactionHop == hop+1 {
			crossing = &cr
		}
	}
	if crossing != nil && crossing.Status != mint.TxStCanceled {
		return false
	}

	return true
}

// PriceRegexp is used to validate and parse a transaction price.
var PriceRegexp = regexp.MustCompile(
	"^([0-9]+)\\/([0-9]+)$")

// ExtractPrice validates a price (pB/pQ).
func ExtractPrice(
	ctx context.Context,
	price string,
) (*big.Int, *big.Int, error) {
	m := PriceRegexp.FindStringSubmatch(price)
	if len(m) == 0 {
		return nil, nil, errors.Trace(errors.Newf("Invalid price: %s", price))
	}
	var basePrice big.Int
	_, success := basePrice.SetString(m[1], 10)
	if !success ||
		basePrice.Cmp(new(big.Int)) < 0 ||
		basePrice.Cmp(model.MaxAssetAmount) >= 0 {
		return nil, nil, errors.Trace(errors.Newf("Invalid price: %s", price))
	}

	var quotePrice big.Int
	_, success = quotePrice.SetString(m[2], 10)
	if !success ||
		quotePrice.Cmp(new(big.Int)) < 0 ||
		quotePrice.Cmp(model.MaxAssetAmount) >= 0 {
		return nil, nil, errors.Trace(errors.Newf("Invalid price: %s", price))
	}

	return &basePrice, &quotePrice, nil
}
