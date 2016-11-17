package endpoint

import (
	"fmt"
	"math/big"
	"net/http"
	"regexp"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
)

const (
	// EndPtCreateOffer creates a new assset.
	EndPtCreateOffer EndPtName = "CreateOffer"
)

func init() {
	registrar[EndPtCreateOffer] = NewCreateOffer
}

// CreateOffer creates a new canonical offer and triggers its propagation to
// all the mints involved.
type CreateOffer struct {
	Client *mint.Client

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

// OfferPriceRegexp is used to validate and parse an offer price.
var OfferPriceRegexp = regexp.MustCompile(
	"^([0-9]+)\\/([0-9]+)$")

// Validate validates the input parameters.
func (e *CreateOffer) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	e.Owner = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		env.Get(ctx).Config[mint.EnvCfgMintHost])

	// Validate asset pair.
	pair, err := mint.AssetResourcesFromPair(ctx, r.PostFormValue("pair"))
	if err != nil {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "pair_invalid",
			"The asset pair you provided is invalid: %s.",
			r.PostFormValue("pair"),
		))
	}
	e.Pair = pair

	// Validate price.
	m := OfferPriceRegexp.FindStringSubmatch(r.PostFormValue("price"))
	if len(m) == 0 {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The offer price you provided is invalid: %s. Prices must have "+
				"the form 'pB/pQ' where pB is the base asset price and pQ "+
				"is the quote asset price.",
			r.PostFormValue("price"),
		))
	}
	var basePrice big.Int
	_, success := basePrice.SetString(m[1], 10)
	if !success ||
		basePrice.Cmp(new(big.Int)) < 0 ||
		basePrice.Cmp(model.MaxAssetAmount) >= 0 {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The base asset price you provided is invalid: %s. Asset prices "+
				"must be integers between 0 and 2^128.",
			m[1],
		))
	}
	e.BasePrice = basePrice

	var quotePrice big.Int
	_, success = quotePrice.SetString(m[2], 10)
	if !success ||
		quotePrice.Cmp(new(big.Int)) < 0 ||
		quotePrice.Cmp(model.MaxAssetAmount) >= 0 {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The quote asset price you provided is invalid: %s. Asset prices "+
				"must be integers between 0 and 2^128.",
			m[2],
		))
	}
	e.QuotePrice = quotePrice

	// Validate amount.
	var amount big.Int
	_, success = amount.SetString(r.PostFormValue("amount"), 10)
	if !success ||
		amount.Cmp(new(big.Int)) < 0 ||
		amount.Cmp(model.MaxAssetAmount) >= 0 {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "amount_invalid",
			"The amount you provided is invalid: %s. Amounts must be "+
				"integers between 0 and 2^128.",
			r.PostFormValue("amount"),
		))
	}
	e.Amount = amount

	return nil
}

// Execute executes the endpoint.
func (e *CreateOffer) Execute(
	r *http.Request,
) (*int, *svc.Resp, error) {
	ctx := r.Context()

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	// Create canonical offer locally.
	offer, err := model.CreateCanonicalOffer(ctx,
		e.Owner, e.Pair[0].Name, e.Pair[1].Name,
		model.Amount(e.BasePrice), model.Amount(e.QuotePrice),
		model.Amount(e.Amount), model.OfStActive)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	// We commit first so that the offer is visible to subsequent requests
	// hitting the mint (from other mint to validate the offer after
	// propagation).
	db.Commit(ctx)

	// TODO: propagate offer to assets' mints, failing silently if
	// unsuccessful.

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"offer": format.JSONPtr(mint.NewOfferResource(ctx, offer)),
	}, nil
}
