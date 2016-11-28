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
type TxPlan struct {
	Actions     []TxAction
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

			// TODO(stan): validate that the offer owner owns the base asset.

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
		Actions:     []TxAction{},
		Transaction: fmt.Sprintf("%s[%s]", tx.Owner, tx.Token),
	}

	// FIRST PASS: consists in computing the actions for all operations,
	// leaving the amounts empty.

	// Generate first action.
	_, host, err := mint.UsernameAndMintHostFromAddress(ctx, tx.Owner)
	if err != nil {
		return nil, errors.Trace(err)
	}
	plan.Actions = append(plan.Actions, TxAction{
		Mint:                 host,
		Owner:                tx.Owner,
		Type:                 TxActTpOperation,
		OperationAsset:       &tx.BaseAsset,
		Amount:               nil, // computed on second pass
		OperationDestination: nil, // computed by next offer
		OperationSource:      &tx.Owner,
	})

	// Generate actions from path of offers.
	for i, offer := range offers {
		offer := offer
		pair, err := mint.AssetResourcesFromPair(ctx, offer.Pair)
		if err != nil {
			return nil, errors.Trace(err)
		}
		// Compare the previous operation asset with the offer quote asset.
		if pair[1].Name != *plan.Actions[2*i].OperationAsset {
			return nil, errors.Trace(errors.Newf(
				"Operation/Offer asset mismatch at offer %s: %s expected %s.",
				offer.ID, pair[0].Name, *plan.Actions[2*i].OperationAsset))
		}
		// Fill the previous operation destination.
		plan.Actions[2*i].OperationDestination = &offer.Owner
		// Add the crossing action.
		_, host, err := mint.UsernameAndMintHostFromAddress(ctx, offer.Owner)
		if err != nil {
			return nil, errors.Trace(err)
		}
		plan.Actions = append(plan.Actions, TxAction{
			Mint:          host,
			Owner:         offer.Owner,
			Type:          TxActTpCrossing,
			CrossingOffer: &offer.ID,
			Amount:        nil, // computed on second pass
		})
		// Add the next operation action.
		_, host, err = mint.UsernameAndMintHostFromAddress(ctx, pair[0].Owner)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if offer.Owner != pair[0].Owner {
			return nil, errors.Trace(errors.Newf(
				"Offer owner (%s) is not the offer base asset owner (%s).",
				offer.Owner, pair[0].Owner))
		}
		plan.Actions = append(plan.Actions, TxAction{
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
	if tx.QuoteAsset != *plan.Actions[len(plan.Actions)-1].OperationAsset {
		return nil, errors.Trace(errors.Newf(
			"Operation/Transaction asset mismatchs: %s expected %s.",
			tx.QuoteAsset, *plan.Actions[len(plan.Actions)-1].OperationAsset))
	}
	// Compute the last operation destination.
	plan.Actions[len(plan.Actions)-1].OperationDestination = &tx.Destination

	// SECOND PASS: consists in computing the amounts for each operations.

	// Compute the last amount, this is the transaction amount.
	plan.Actions[len(plan.Actions)-1].Amount = (*big.Int)(&tx.Amount)

	// Compute amounts for each action.
	for i := len(offers) - 1; i >= 0; i-- {
		// Offer amounts are expressed in quote asset
		basePrice, quotePrice, err := ValidatePrice(ctx, offers[i].Price)
		if err != nil {
			return nil, errors.Trace(err)
		}
		amount := new(big.Int).Mul(plan.Actions[2*(i+1)].Amount, basePrice)
		amount, remainder := new(big.Int).QuoRem(
			amount, quotePrice, new(big.Int))

		// Transactions do cross offers on non congruent prices, costing one
		// base unit of quote asset. If the difference of scale between assets
		// is high, this can cost a lot to the owner of the transaction (but if
		// they issued it, they know).
		if remainder.Cmp(big.NewInt(0)) > 0 {
			amount = new(big.Int).Add(amount, big.NewInt(1))
		}

		plan.Actions[2*i].Amount = amount
		plan.Actions[2*i+1].Amount = amount

		if amount.Cmp(offers[i].Remainder) > 0 {
			return nil, errors.Trace(errors.Newf(
				"Insufficient remainder for offer %s: %s but needs %s.",
				offers[i].ID, offers[i].Remainder.String(),
				amount.String()))
		}
	}

	logLine := fmt.Sprintf("Transaction plan for %s:", plan.Transaction)
	for i, a := range plan.Actions {
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
				*a.CrossingOffer, offers[i/2].Pair, offers[i/2].Price)
		}
	}
	mint.Logf(ctx, logLine)

	return &plan, nil
}

// Check checks that the plan was properly executed at the specified hop by
// retrieving the transaction ont that mint and checking the action against the
// advertised operations and crossings.
func (p *TxPlan) Check(
	ctx context.Context,
	client *mint.Client,
	hop int8,
) error {
	action := p.Actions[hop]
	transaction, err := client.RetrieveTransaction(ctx,
		p.Transaction, &action.Mint)
	if err != nil {
		return errors.Trace(err)
	}

	switch action.Type {
	case TxActTpOperation:
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
		if operation.Owner != action.Owner {
			return errors.Newf("Operation at hop %d owner mismatch: "+
				"%s expected %s",
				hop, operation.Owner, action.Owner)
		}
		if operation.Amount.Cmp(action.Amount) != 0 {
			return errors.Newf("Operation at hop %d amount mismatch: "+
				"%s expected %s",
				hop, operation.Amount.String(), action.Amount.String())
		}
		if operation.Asset != *action.OperationAsset {
			return errors.Newf("Operation at hop %d asset mismatch: "+
				"%s expected %s",
				hop, operation.Asset, *action.OperationAsset)
		}
		if operation.Source != *action.OperationSource {
			return errors.Newf("Operation at hop %d source mismatch: "+
				"%s expected %s",
				hop, operation.Source, *action.OperationSource)
		}
		if operation.Destination != *action.OperationDestination {
			return errors.Newf("Operation at hop %d destination mismatch: "+
				"%s expected %s",
				hop, operation.Destination, *action.OperationDestination)
		}
	case TxActTpCrossing:
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
		if crossing.Owner != action.Owner {
			return errors.Newf("Crossing at hop %d owner mismatch: "+
				"%s expected %s",
				hop, crossing.Owner, action.Owner)
		}
		if crossing.Amount.Cmp(action.Amount) != 0 {
			return errors.Newf("Crossing at hop %d amount mismatch: "+
				"%s expected %s",
				hop, crossing.Amount.String(), action.Amount.String())
		}
		if crossing.Offer != *action.CrossingOffer {
			return errors.Newf("Crossing at hop %d offer mismatch: "+
				"%s expected %s",
				hop, crossing.Offer, *action.CrossingOffer)
		}
	}

	return nil
}
