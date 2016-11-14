package mint

import (
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/respond"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
	"goji.io/pat"
)

type controller struct {
	client *Client
}

// CreateAsset controls the creation of new assets.
func (c *controller) CreateAsset(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	userToken := authentication.Get(ctx).User.Token

	code := r.PostFormValue("code")
	if !model.AssetCodeRegexp.MatchString(code) {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
			400, "code_invalid",
			"The asset code provided is invalid: %s. Asset codes can use "+
				"alphanumeric upercased and `-` characters only.",
			code,
		)))
		return
	}

	scale, err := strconv.ParseInt(r.PostFormValue("scale"), 10, 8)
	if err != nil ||
		(int8(scale) < model.AssetMinScale ||
			int8(scale) > model.AssetMaxScale) {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "scale_invalid",
			"The asset scale provided is invalid: %s. Asset scales must be "+
				"integers between %d and %d.",
			r.PostFormValue("scale"), model.AssetMinScale, model.AssetMaxScale,
		)))
		return
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	asset, err := model.CreateAsset(ctx, userToken, code, int8(scale))
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case model.ErrUniqueConstraintViolation:
			respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
				400, "asset_already_exists",
				"You already created an asset with the same code: %s.",
				code,
			)))
		default:
			respond.Error(ctx, w, errors.Trace(err)) // 500
		}
		return
	}

	db.Commit(ctx)

	respond.Created(ctx, w, svc.Resp{
		"asset": format.JSONPtr(NewAssetResource(ctx,
			asset, authentication.Get(ctx).User,
			env.Get(ctx).Config[EnvCfgMintHost])),
	})
}

// CreateOperation creates a new operation that moves asset from a source
// balance to a destination balance:
// - no `source` specified: issuance.
// - no `destination` specified: annihilation.
// - both specified: transfer from a balance to another.
// Only the asset creator can create operation on an asset. To transfer money,
// users owning an asset whould use transactions.
func (c *controller) CreateOperation(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	// Validate asset.
	a, err := AssetResourceFromName(ctx, pat.Param(r, "asset"))
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "asset_invalid",
			"The asset name you provided is invalid: %s.",
			pat.Param(r, "asset"),
		)))
		return
	}
	username, host, err := UsernameAndMintHostFromAddress(
		ctx, a.Issuer)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "asset_invalid",
			"The asset name you provided has an invalid issuer address: %s.",
			a.Issuer,
		)))
		return
	}

	// Validate that the issuer is attempting to create the operation.
	if host != env.Get(ctx).Config[EnvCfgMintHost] ||
		username != authentication.Get(ctx).User.Username {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
			400, "operation_not_authorized",
			"You can only create operations for assets created by the "+
				"account you are currently authenticated with: %s@%s. This "+
				"operation's asset was created by: %s@%s. If you own %s, "+
				"and wish to transfer some of it, you should create a "+
				"transaction directly from your mint instead.",
			authentication.Get(ctx).User.Username,
			env.Get(ctx).Config[EnvCfgMintHost],
			username, host, a.Name,
		)))
		return
	}

	// Validate amount.
	var amount big.Int
	_, success := amount.SetString(r.PostFormValue("amount"), 10)
	if !success ||
		amount.Cmp(new(big.Int)) < 0 ||
		amount.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "amount_invalid",
			"The amount you provided is invalid: %s. Amounts must be "+
				"integers between 0 and 2^128.",
			r.PostFormValue("amount"),
		)))
		return
	}

	// Validate source
	var srcAddress *string
	if r.PostFormValue("source") != "" {
		addr, err := NormalizedAddress(ctx, r.PostFormValue("source"))
		if err != nil {
			respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
				400, "source_invalid",
				"The source address you provided is invalid: %s.",
				*srcAddress,
			)))
			return
		}
		srcAddress = &addr
	}

	// Validate destination
	var dstAddress *string
	if r.PostFormValue("destination") != "" {
		addr, err := NormalizedAddress(ctx, r.PostFormValue("destination"))
		if err != nil {
			respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
				400, "destination_invalid",
				"The destination address you provided is invalid: %s.",
				*dstAddress,
			)))
			return
		}
		dstAddress = &addr
	}

	if srcAddress == nil && dstAddress == nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "operation_invalid",
			"The operation has no source and no destination. You must "+
				"specify at least one of them (no source: issuance; no "+
				"destination: annihilation; source and destination: "+
				"transfer).",
		)))
		return
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	asset, err := model.LoadAssetByIssuerCodeScale(ctx,
		authentication.Get(ctx).User.Token, a.Code, a.Scale)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	} else if asset == nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
			400, "asset_not_found",
			"The asset you are trying to operate does not exist: %s. Try "+
				"creating it first.",
			a.Name,
		)))
		return
	}

	var srcBalance *model.Balance
	if srcAddress != nil {
		srcBalance, err = model.LoadBalanceByAssetOwner(ctx,
			asset.Token, *srcAddress)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return
		} else if srcBalance == nil {
			respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
				400, "source_invalid",
				"The source address you provided has no existing balance: %s.",
				*srcAddress,
			)))
			return
		}
	}

	var dstBalance *model.Balance
	if dstAddress != nil {
		dstBalance, err = model.LoadOrCreateBalanceByAssetOwner(ctx,
			asset.Token, *dstAddress)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return
		}
	}

	operation, err := model.CreateOperation(ctx,
		asset.Token, srcAddress, dstAddress, model.Amount(amount))
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	if dstBalance != nil {
		(*big.Int)(&dstBalance.Value).Add(
			(*big.Int)(&dstBalance.Value), (*big.Int)(&operation.Amount))

		// Checks if the dstBalance is positive and not overflown.
		b := (*big.Int)(&dstBalance.Value)
		if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
			b.Cmp(new(big.Int)) < 0 {
			respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
				400, "amount_invalid",
				"The resulting destination balance is invalid: %s. The "+
					"balance must be an integer between 0 and 2^128.",
				b.String(),
			)))
			return
		}

		err = dstBalance.Save(ctx)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return
		}
	}

	if srcBalance != nil {
		(*big.Int)(&srcBalance.Value).Sub(
			(*big.Int)(&srcBalance.Value), (*big.Int)(&operation.Amount))

		// Checks if the srcBalance is positive and not overflown.
		b := (*big.Int)(&srcBalance.Value)
		if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
			b.Cmp(new(big.Int)) < 0 {
			respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
				400, "amount_invalid",
				"The resulting source balance is invalid: %s. The "+
					"balance must be an integer between 0 and 2^128.",
				b.String(),
			)))
			return
		}

		err = srcBalance.Save(ctx)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return
		}
	}

	db.Commit(ctx)

	respond.OK(ctx, w, svc.Resp{
		"operation": format.JSONPtr(NewOperationResource(ctx,
			operation,
			NewAssetResource(ctx,
				asset, authentication.Get(ctx).User,
				env.Get(ctx).Config[EnvCfgMintHost]))),
	})
}

// RetrieveOffer retrieves an offer based on its id. It is not authenticated
// and is used to verify offers when they get propagated.
func (c *controller) RetrieveOffer(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	id := pat.Param(r, "offer")

	// Validate id.
	address, token, err := NormalizedAddressAndTokenFromID(ctx, id)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "id_invalid",
			"The offer id you provided is invalid: %s. Offer ids must have "+
				"the form kgodel@princeton.edu[offer_*]",
			id,
		)))
		return
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	offer, err := model.LoadOfferByToken(ctx, token)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	} else if offer == nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
			404, "offer_not_found",
			"The offer you are trying to retrieve does not exist: %s.",
			id,
		)))
		return
	}

	if offer.Owner != address {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
			404, "offer_not_found",
			"The offer you are trying to retrieve does not exist: %s.",
			id,
		)))
	}

	db.Commit(ctx)

	respond.OK(ctx, w, svc.Resp{
		"offer": format.JSONPtr(NewOfferResource(ctx, offer)),
	})
}

// CreateOffer routes the offer creation based on authentication. Initial
// authenticated offer creation calls into CreateCanonicalOffer, while
// non-authenticated cross-mint offer propagation calls into
// CreateOfferPropagation.
func (c *controller) CreateOffer(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	switch authentication.Get(ctx).Status {
	case authentication.AutStSucceeded:
		c.CreateCanonicalOffer(w, r)
	case authentication.AutStSkipped:
		c.CreatePropagatedOffer(w, r)
	default:
		respond.Error(ctx, w, errors.Trace(errors.Newf(
			"Unexpected authentication status for offer creation: %s",
			authentication.Get(ctx).Status),
		)) // 500
	}
}

// OfferPriceRegexp is used to validate and parse an offer price.
var OfferPriceRegexp = regexp.MustCompile(
	"^([0-9]+)\\/([0-9]+)$")

// CreateCanonicalOffer creates a new canonical offer. Offer creation involves
// contacting the mints for the offer's assets and storing the canonical
// version of the offer locally.
func (c *controller) CreateCanonicalOffer(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	owner := fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		env.Get(ctx).Config[EnvCfgMintHost])

	// Validate asset pair.
	pair, err := AssetResourcesFromPair(ctx, r.PostFormValue("pair"))
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "pair_invalid",
			"The asset pair you provided is invalid: %s.",
			r.PostFormValue("pair"),
		)))
		return
	}

	// Validate price.
	m := OfferPriceRegexp.FindStringSubmatch(r.PostFormValue("price"))
	if len(m) == 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The offer price you provided is invalid: %s. Prices must have "+
				"the form 'pB/pQ' where pB is the base asset price and pQ "+
				"is the quote asset price.",
			r.PostFormValue("price"),
		)))
		return
	}
	var basePrice big.Int
	_, success := basePrice.SetString(m[1], 10)
	if !success ||
		basePrice.Cmp(new(big.Int)) < 0 ||
		basePrice.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The base asset price you provided is invalid: %s. Asset prices "+
				"must be integers between 0 and 2^128.",
			m[1],
		)))
		return
	}
	var quotePrice big.Int
	_, success = quotePrice.SetString(m[2], 10)
	if !success ||
		quotePrice.Cmp(new(big.Int)) < 0 ||
		quotePrice.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The quote asset price you provided is invalid: %s. Asset prices "+
				"must be integers between 0 and 2^128.",
			m[2],
		)))
		return
	}

	// Validate amount.
	var amount big.Int
	_, success = amount.SetString(r.PostFormValue("amount"), 10)
	if !success ||
		amount.Cmp(new(big.Int)) < 0 ||
		amount.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "amount_invalid",
			"The amount you provided is invalid: %s. Amounts must be "+
				"integers between 0 and 2^128.",
			r.PostFormValue("amount"),
		)))
		return
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	// Create canonical offer locally.
	offer, err := model.CreateCanonicalOffer(ctx,
		owner, pair[0].Name, pair[1].Name,
		model.Amount(basePrice), model.Amount(quotePrice),
		model.Amount(amount), model.OfStActive)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	// We commit first so that the offer is visible to subsequent requests
	// hitting the mint (from other mint to validate the offer after
	// propagation).
	db.Commit(ctx)

	// TODO: propagate offer to assets' mints, failing silently if
	// unsuccessful.

	respond.OK(ctx, w, svc.Resp{
		"offer": format.JSONPtr(NewOfferResource(ctx, offer)),
	})
}

// CreatePropagatedOffer creates a new offer through propagation. Propagation
// is validated by contacting the mint of the offer's owner and stored locally.
func (c *controller) CreatePropagatedOffer(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	// Validate id.
	id := r.PostFormValue("id")
	owner, token, err := NormalizedAddressAndTokenFromID(ctx, id)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "id_invalid",
			"The offer id you provided is invalid: %s. Offer ids must have "+
				"the form kgodel@princeton.edu[offer_*]",
			id,
		)))
		return
	}
	_, host, err := UsernameAndMintHostFromAddress(ctx, owner)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "id_invalid",
			"The offer id you provided has an invalid owner: %s.",
			owner,
		)))
		return
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	// Check if the offer exists locally
	offer, err := model.LoadOfferByToken(ctx, token)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	// If the mint owns this offer, just return
	if offer != nil && offer.Type == model.OfTpCanonical {
		respond.OK(ctx, w, svc.Resp{
			"offer": format.JSONPtr(NewOfferResource(ctx, offer)),
		})
		return
	}

	// Check that the offer exists and the mint is reachable.
	o, err := c.client.RetrieveOffer(ctx, id)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			402, "canonical_offer_cannot_be_retrieved",
			"The canonical offer %s could not be retrieved. This might mean "+
				"that mint %s is not reachable from this mint or can be due "+
				"to the fact that %s is not valid anymore on %s.",
			id, host, id, host,
		)))
		return
	}

	// Validate asset pair.
	pair, err := AssetResourcesFromPair(ctx, o.Pair)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			402, "retrieved_pair_invalid",
			"The asset pair of the offer retrieved is invalid: %s.",
			o.Pair,
		)))
		return
	}

	// Validate price.
	m := OfferPriceRegexp.FindStringSubmatch(o.Price)
	if len(m) == 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			402, "retrieved_price_invalid",
			"The offer price of the retrieved offer is invalid: %s.",
			o.Price,
		)))
		return
	}
	var basePrice big.Int
	_, success := basePrice.SetString(m[1], 10)
	if !success ||
		basePrice.Cmp(new(big.Int)) < 0 ||
		basePrice.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			402, "retrieved_price_invalid",
			"The base asset price of the retrieved offer is invalid: %s.",
			m[1],
		)))
		return
	}
	var quotePrice big.Int
	_, success = quotePrice.SetString(m[2], 10)
	if !success ||
		quotePrice.Cmp(new(big.Int)) < 0 ||
		quotePrice.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The quote asset price of the retrieved offer is invalid: %s.",
			m[2],
		)))
		return
	}

	if offer != nil {
		offer.Owner = owner
		offer.BaseAsset = pair[0].Name
		offer.QuoteAsset = pair[1].Name
		offer.BasePrice = model.Amount(basePrice)
		offer.QuotePrice = model.Amount(quotePrice)
		offer.Amount = model.Amount(*o.Amount)
		offer.Status = model.OfStatus(o.Status)

		err = offer.Save(ctx)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return
		}
	} else {
		// Create non-canonical offer locally.
		offer, err = model.CreatePropagatedOffer(ctx,
			token, time.Unix(0, o.Created*1000*1000), owner, pair[0].Name, pair[1].Name,
			model.Amount(basePrice), model.Amount(quotePrice),
			model.Amount(*o.Amount), model.OfStActive)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return
		}
	}

	respond.OK(ctx, w, svc.Resp{
		"offer": format.JSONPtr(NewOfferResource(ctx, offer)),
	})
}
