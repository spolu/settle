package mint

import (
	"math/big"
	"net/http"
	"strconv"

	"goji.io/pat"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/respond"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/lib/tx"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/model"

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
// Only the asset creator can create operation on an asset.
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
				"operation's asset was created by: %s@%s.",
			authentication.Get(ctx).User.Username, c.mintHost,
			username, host,
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
			"The amount you provided is invalid: %s. Amount must be a "+
				"an integer between 0 and 2^128.",
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
		srcBalance, err = model.LoadOrCreateBalanceByAssetOwner(ctx,
			asset.Token, *srcAddress)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
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
		asset.Token, srcAddress, dstAddress, model.BigInt(amount))
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
