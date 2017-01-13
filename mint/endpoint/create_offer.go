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
			400, "not_authorized",
			"You can only create offers whose base asset is owned by the "+
				"account you are currently authenticated with: %s. This "+
				"offer base asset was created by: %s.",
			e.Owner, e.Pair[0].Owner,
		))
	}

	if e.Pair[0].Name == e.Pair[1].Name {
		return errors.Trace(errors.NewUserErrorf(nil,
			400, "pair_invalid",
			"You cannot create an offer with the same base and quote asset.",
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

	return nil
}

// Execute executes the endpoint.
func (e *CreateOffer) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	// Validate that the asset exists locally.
	asset, err := model.LoadCanonicalAssetByName(ctx, e.Pair[0].Name)
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
	of, err := model.CreateCanonicalOffer(ctx,
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

	mint.Logf(ctx,
		"Created offer: id=%s[%s] created=%q propagation=%s "+
			"base_asset=%s quote_asset=%s base_price=%s quote_price=%s "+
			"amount=%s status=%s remainder=%s",
		of.Owner, of.Token, of.Created, of.Propagation, of.BaseAsset,
		of.QuoteAsset,
		(*big.Int)(&of.BasePrice).String(), (*big.Int)(&of.QuotePrice),
		(*big.Int)(&of.Amount).String(), of.Status,
		(*big.Int)(&of.Remainder).String())

	err = async.Queue(ctx, task.NewPropagateOffer(ctx, time.Now(), of.ID()))
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"offer": format.JSONPtr(model.NewOfferResource(ctx, of)),
	}, nil
}
