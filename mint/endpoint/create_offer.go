// OWNER: stan

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
	"github.com/spolu/settle/mint/model"
)

const (
	// EndPtCreateOffer creates a new offer.
	EndPtCreateOffer EndPtName = "CreateOffer"
)

func init() {
	registrar[EndPtCreateOffer] = NewCreateOffer
}

// CreateOffer creates a new canonical offer and triggers its propagation to
// all the mints involved. Offer are represented as asks: base asset (left) is
// offered in exchange for quote asset (right) for specified amount (of quote
// asset) at specified price.
type CreateOffer struct {
	Client *mint.Client

	ID         string // propagation
	Token      string // propagation
	Owner      string
	Pair       []mint.AssetResource
	BasePrice  big.Int
	QuotePrice big.Int
	Amount     big.Int
}

// NewCreateOffer constructs and initialiezes the endpoint.
func NewCreateOffer(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}
	return &CreateOffer{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *CreateOffer) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	switch authentication.Get(ctx).Status {
	case authentication.AutStSkipped:
		// Validate id.
		id, owner, token, err := ValidateID(ctx, pat.Param(r, "offer"))
		if err != nil {
			return errors.Trace(err)
		}
		e.ID = *id
		e.Owner = *owner
		e.Token = *token

	case authentication.AutStSucceeded:
		e.Owner = fmt.Sprintf("%s@%s",
			authentication.Get(ctx).User.Username, mint.GetHost(ctx))

		// Validate asset pair.
		pair, err := ValidateAssetPair(ctx, r.PostFormValue("pair"))
		if err != nil {
			return errors.Trace(err) // 400
		}
		e.Pair = pair

		// Validate that the base asset's owner matches the offer owner
		if e.Pair[0].Owner != e.Owner {
			return errors.Trace(errors.NewUserErrorf(nil,
				400, "offer_not_authorized",
				"You can only create offers whose base asset is owned by the "+
					"account you are currently authenticated with: %s. This "+
					"offer base asset was created by: %s.",
				e.Owner, e.Pair[0].Owner,
			))
		}

		// Validate price.
		basePrice, quotePrice, err := ValidatePrice(ctx, r.PostFormValue("price"))
		if err != nil {
			return errors.Trace(err) // 400
		}
		e.BasePrice = *basePrice
		e.QuotePrice = *quotePrice

		// Validate amount.
		amount, err := ValidateAmount(ctx, r.PostFormValue("amount"))
		if err != nil {
			return errors.Trace(err) // 400
		}
		e.Amount = *amount
	}

	return nil
}

// Execute executes the endpoint.
func (e *CreateOffer) Execute(
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

// ExecuteCanonical executes the canonical creation of the offer (owner mint).
func (e *CreateOffer) ExecuteCanonical(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	// Validate that the asset exists locally.
	asset, err := model.LoadAssetByName(ctx, e.Pair[0].Name)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if asset == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "asset_not_found",
			"The base asset you specifed does not exist: %s. You must create "+
				"an asset before you use it as base asset of an offer.",
			e.Pair[0].Name,
		))
	}

	// Create canonical offer locally.
	offer, err := model.CreateCanonicalOffer(ctx,
		e.Owner,
		e.Pair[0].Name,
		e.Pair[1].Name,
		model.Amount(e.BasePrice),
		model.Amount(e.QuotePrice),
		model.Amount(e.Amount),
		mint.OfStActive,
		model.Amount(e.Amount),
	)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	of := model.NewOfferResource(ctx, offer)

	err = async.Queue(ctx, task.NewPropagateOffer(ctx, of.ID))
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"offer": format.JSONPtr(of),
	}, nil
}

// ExecutePropagated executes the propagation of an offer (involved mint).
func (e *CreateOffer) ExecutePropagated(
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

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	code := http.StatusCreated

	of, err := model.LoadPropagatedOfferByOwnerToken(ctx, owner, token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if of != nil {
		// Only the offer status and remainder are mutable.
		of.Status = offer.Status
		of.Remainder = model.Amount(*remainder)

		// TODO(stan): check that the rest offer hasn't changed.

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
	}

	db.Commit(ctx)

	return ptr.Int(code), &svc.Resp{
		"offer": format.JSONPtr(model.NewOfferResource(ctx, of)),
	}, nil
}
