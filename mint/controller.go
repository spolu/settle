package mint

import (
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strconv"

	"goji.io/pat"

	"github.com/spolu/peer-currencies/lib/errors"
	"github.com/spolu/peer-currencies/lib/format"
	"github.com/spolu/peer-currencies/lib/respond"
	"github.com/spolu/peer-currencies/lib/svc"
	"github.com/spolu/peer-currencies/lib/tx"
	"github.com/spolu/peer-currencies/mint/lib/authentication"
	"github.com/spolu/peer-currencies/model"

	"golang.org/x/net/context"
)

type controller struct {
	mintHost string
	client   *Client
}

// CreateAsset controls the creation of new assets.
func (c *controller) CreateAsset(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
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
			400, "username_invalid",
			"The asset scale provided is invalid: %s. Asset scales must be "+
				"integers between %d and %d.",
			r.PostFormValue("scale"), model.AssetMinScale, model.AssetMaxScale,
		)))
		return
	}

	ctx = tx.Begin(ctx, model.MintDB())
	defer tx.LoggedRollback(ctx)

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

	tx.Commit(ctx)

	respond.Success(ctx, w, svc.Resp{
		"asset": format.JSONPtr(NewAssetResource(ctx,
			asset, authentication.Get(ctx).User, c.mintHost)),
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
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	// Validate asset.
	a, err := AssetResourceFromName(
		ctx,
		pat.Param(ctx, "asset"),
	)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "asset_invalid",
			"The asset name you provided is invalid: %s.",
			pat.Param(ctx, "asset"),
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
	if host != c.mintHost || username != authentication.Get(ctx).User.Username {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
			400, "operation_not_authorized",
			"You can only create operations for assets created by the "+
				"account you are currently authenticated with: %s@%s. This "+
				"operation's asset was created by: %s@%s. If you own %s, "+
				"and wish to transfer some of it, you should create a "+
				"transaction directly from your mint instead.",
			authentication.Get(ctx).User.Username, c.mintHost,
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

	ctx = tx.Begin(ctx, model.MintDB())
	defer tx.LoggedRollback(ctx)

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

	tx.Commit(ctx)

	respond.Success(ctx, w, svc.Resp{
		"operation": format.JSONPtr(NewOperationResource(ctx,
			operation,
			NewAssetResource(ctx,
				asset, authentication.Get(ctx).User, c.mintHost))),
	})
}

// RetrieveOffer retrieves an offer based on its id. It is not authenticated
// and is used to verify offers when they get propagated.
func (c *controller) RetrieveOffer(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	id := pat.Param(ctx, "offer")

	// Validate id.
	address, token, err := NormalizedAddressAndTokenFromID(ctx, id)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "id_invalid",
			"The offer id you provided is invalid: %s.",
			id,
		)))
		return
	}

	ctx = tx.Begin(ctx, model.MintDB())
	defer tx.LoggedRollback(ctx)

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

	tx.Commit(ctx)

	respond.Success(ctx, w, svc.Resp{
		"offer": format.JSONPtr(NewOfferResource(ctx, offer)),
	})
}

// CreateOffer routes the offer creation based on authentication. Initial
// authenticated offer creation calls into CreateInitialOffer, while
// non-authenticated cross-mint offer propagation calls into
// CreateOfferPropagation.
func (c *controller) CreateOffer(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	switch authentication.Get(ctx).Status {
	case authentication.AutStSucceeded:
		c.CreateInitialOffer(ctx, w, r)
	case authentication.AutStSkipped:
		c.CreateOfferPropagation(ctx, w, r)
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

// CreateInitialOffer creates a new initial offer. Offer creation involves
// contacting the mints for the offer's assets and storing the canonical
// version of the offer locally.
func (c *controller) CreateInitialOffer(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	// Validate asset pair.
	pair, err := AssetResourcesFromPair(
		ctx,
		r.PostFormValue("pair"),
	)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "pair_invalid",
			"The asset pair you provided is invalid: %s.",
			r.PostFormValue("pair"),
		)))
		return
	}

	// Validate bid.
	var oftype model.OfType
	switch r.PostFormValue("type") {
	case string(model.OfTpBid):
		oftype = model.OfTpBid
	case string(model.OfTpAsk):
		oftype = model.OfTpAsk
	default:
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "type_invalid",
			"The offer type you provided is invalid: %s. Accepted values are "+
				"bid, ask.",
			r.PostFormValue("type"),
		)))
	}

	// Validate price.
	m := OfferPriceRegexp.FindStringSubmatch(r.PostFormValue("price"))
	if len(m) == 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The offer price you provided is invalid: %s. Prices must have "+
				"the form \"pB/pQ\" where pB is the base asset price and pQ "+
				"is the quote asset price.",
			r.PostFormValue("type"),
		)))
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
	_, success = quotePrice.SetString(m[1], 10)
	if !success ||
		quotePrice.Cmp(new(big.Int)) < 0 ||
		quotePrice.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "price_invalid",
			"The quote asset price you provided is invalid: %s. Asset prices "+
				"must be integers between 0 and 2^128.",
			m[1],
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

	owner := fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username, c.mintHost)

	ctx = tx.Begin(ctx, model.MintDB())
	defer tx.LoggedRollback(ctx)

	// Create canonical offer locally.
	offer, err := model.CreateOffer(ctx,
		owner, pair[0].Name, pair[1].Name, oftype, model.Amount(basePrice),
		model.Amount(quotePrice), model.Amount(amount), model.OfStActive)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	// We commit first so that the offer is visible to subsequent requests
	// hitting the mint (from other mint to validate the offer after
	// propagation).
	tx.Commit(ctx)

	// TODO: propagate offer to assets' mints, failing silently if
	// unsuccessful.

	respond.Success(ctx, w, svc.Resp{
		"offer": format.JSONPtr(NewOfferResource(ctx, offer)),
	})
}

// CreateOfferPropagation creates a new offer through propagation. Propagation
// is validated by contacting the mint of the offer's owner and stored locally.
func (c *controller) CreateOfferPropagation(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}
