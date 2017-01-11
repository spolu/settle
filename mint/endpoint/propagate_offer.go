package endpoint

import (
	"context"
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
	"github.com/spolu/settle/mint/model"
)

const (
	// EndPtPropagateOffer creates a new offer.
	EndPtPropagateOffer EndPtName = "PropagateOffer"
)

func init() {
	registrar[EndPtPropagateOffer] = NewPropagateOffer
}

// PropagateOffer retrieves a canonical offer and creates a local propagated
// copy of it.
type PropagateOffer struct {
	Client *mint.Client

	ID    string // propagation
	Token string // propagation
	Owner string
}

// NewPropagateOffer constructs and initialiezes the endpoint.
func NewPropagateOffer(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}
	return &PropagateOffer{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *PropagateOffer) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "offer"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Owner = *owner
	e.Token = *token

	return nil
}

// Execute executes the endpoint.
func (e *PropagateOffer) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	offer, err := e.Client.RetrieveOffer(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "propagation_failed",
			"Failed to retrieve canonical offer: %s", e.ID,
		))
	}

	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, offer.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid offer id: %s", offer.ID,
		))
	}

	if e.ID != offer.ID {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected offer id: %s expected %s", offer.ID, e.ID,
		))
	}
	if e.Owner != owner || offer.Owner != owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected offer owner: %s expected %s", owner, e.Owner,
		))
	}
	if e.Token != token {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected offer token: %s expected %s", token, e.Token,
		))
	}

	pair, err := mint.AssetResourcesFromPair(ctx, offer.Pair)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid offer pair: %s", offer.Pair,
		))
	}
	if pair[0].Owner != offer.Owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Operation and pair asset owner mismatch: %s expected %s",
			pair[0].Owner, offer.Owner,
		))
	}

	basePrice, quotePrice, err := ValidatePrice(ctx, offer.Price)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid offer price: %s", offer.Price,
		))
	}
	amount, err := ValidateAmount(ctx, offer.Amount.String())
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid offer amount: %s", offer.Amount.String(),
		))
	}

	switch offer.Status {
	case mint.OfStActive, mint.OfStClosed, mint.OfStConsumed:
	default:
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid offer status: %s", offer.Status,
		))
	}
	remainder, err := ValidateAmount(ctx, offer.Remainder.String())
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid offer remainder: %s", offer.Remainder.String(),
		))
	}

	user, host, err := mint.UsernameAndMintHostFromAddress(ctx, pair[1].Owner)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid pair asset owner: %s", pair[1].Owner,
		))
	}
	if host != mint.GetHost(ctx) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received offer with no impact on any of this mint users.",
		))
	}

	u, err := model.LoadUserByUsername(ctx, user)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if u == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"User impacted by offer does not exist: %s@%s",
			user, mint.GetHost(ctx),
		))
	}

	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	code := http.StatusCreated

	of, err := model.LoadPropagatedOfferByOwnerToken(ctx, owner, token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if of != nil {
		// Only the offer status and remainder are mutable.
		of.Status = offer.Status
		of.Remainder = model.Amount(*remainder)

		err := of.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
		code = http.StatusOK
	} else {
		// Create propagated offer locally.
		of, err = model.CreatePropagatedOffer(ctx,
			owner,
			token,
			time.Unix(0, offer.Created*mint.TimeResolutionNs),
			pair[0].Name,
			pair[1].Name,
			model.Amount(*basePrice),
			model.Amount(*quotePrice),
			model.Amount(*amount),
			offer.Status,
			model.Amount(*remainder),
		)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}

		mint.Logf(ctx,
			"Propagated offer: id=%s[%s] created=%q propagation=%s "+
				"base_asset=%s quote_asset=%s base_price=%s quote_price=%s "+
				"amount=%s status=%s remainder=%s",
			of.Owner, of.Token, of.Created, of.Propagation, of.BaseAsset,
			of.QuoteAsset, of.BasePrice, of.QuotePrice,
			(*big.Int)(&of.Amount).String(), of.Status,
			(*big.Int)(&of.Remainder).String())
	}

	db.Commit(ctx)

	return ptr.Int(code), &svc.Resp{
		"offer": format.JSONPtr(model.NewOfferResource(ctx, of)),
	}, nil
}
