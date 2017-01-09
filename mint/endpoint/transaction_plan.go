// OWNER: stan

package endpoint

import (
	"context"
	"fmt"
	"math/big"

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

// ComputePlan retrieves the offers of the path and compute the transaction
// plan.
func ComputePlan(
	ctx context.Context,
	client *mint.Client,
	tx *model.Transaction,
) (*TxPlan, error) {
	g, ctx := errgroup.WithContext(ctx)
	offers := make([]mint.OfferResource, len(tx.Path))

	for i, id := range tx.Path {
		i, id := i, id
		g.Go(func() error {
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
		Transaction: fmt.Sprintf("%s[%s]", tx.Owner, tx.Token),
	}

	// FIRST PASS: consists in computing the actions for all hops, leaving the
	// amounts empty.

	bAsset, err := mint.AssetResourceFromName(ctx, tx.BaseAsset)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Generate first hop which has one operation on the mint of the base
	// asset.
	_, host, err := mint.UsernameAndMintHostFromAddress(ctx, bAsset.Owner)
	if err != nil {
		return nil, errors.Trace(err)
	}

	plan.Hops = append(plan.Hops, &TxHop{
		Mint: host,
		OpAction: &TxAction{
			Owner:                bAsset.Owner,
			Type:                 TxActTpOperation,
			OperationAsset:       &bAsset.Name,
			Amount:               nil, // computed on second pass
			OperationDestination: nil, // computed by next offer
			OperationSource:      &tx.Owner,
		},
	})

	// Generate actions from path of offers.
	for i, offer := range offers {
		hop := i + 1
		offer := offer
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
		_, host, err := mint.UsernameAndMintHostFromAddress(ctx, offer.Owner)
		if err != nil {
			return nil, errors.Trace(err)
		}
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
		hop := i + 1
		// Offer amounts are expressed in quote asset
		basePrice, quotePrice, err := ValidatePrice(ctx, offers[i].Price)
		if err != nil {
			return nil, errors.Trace(err)
		}
		amount := new(big.Int).Mul(
			plan.Hops[hop].OpAction.Amount,
			basePrice)
		amount, remainder := new(big.Int).QuoRem(
			amount, quotePrice, new(big.Int))

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
				offers[i-1].Pair, offers[i-1].Price)
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
// transaction plan.
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
