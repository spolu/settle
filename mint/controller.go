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

// CreateAsset controls the creation of new Assets.
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

// IssueAsset controls the issuance of an asset.
func (c *controller) IssueAsset(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
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
			400, "address_invalid",
			"The asset name you provided has an invalid issuer address: %s.",
			a.Issuer,
		)))
		return
	}

	if host != c.mintHost || username != authentication.Get(ctx).User.Username {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(nil,
			400, "issuance_not_authorized",
			"You can only issue assets created by the account you are "+
				"currently authenticated with: %s@%s. Try creating an "+
				"asset and then issuing it.",
			authentication.Get(ctx).User.Username, c.mintHost,
		)))
		return
	}

	var amount big.Int
	_, success := amount.SetString(r.PostFormValue("amount"), 10)
	if !success || amount.Cmp(new(big.Int)) <= 0 || amount.Cmp(model.MaxAssetAmount) >= 0 {
		respond.Error(ctx, w, errors.Trace(errors.NewUserErrorf(err,
			400, "amount_invalid",
			"The amount you provided is invalid: %s. Amount must be a "+
				"positive integer smaller than 2^128.",
			r.PostFormValue("amount"),
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
			"The asset you are trying to issue does not exist: %s. Try "+
				"creating it first.",
			a.Name,
		)))
		return
	}

	// a.Issuer was already normalized (removed `+..@`).
	balance, err := model.LoadOrCreateBalanceByAssetOwner(ctx,
		asset.Token, a.Issuer)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	operation, err := model.CreateOperation(ctx,
		asset.Token, nil, a.Issuer, model.BigInt(amount))
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	(*big.Int)(&balance.Value).Add(
		(*big.Int)(&balance.Value), (*big.Int)(&operation.Amount))
	err = balance.Save(ctx)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	tx.Commit(ctx)

	respond.Success(ctx, w, svc.Resp{
		"operation": format.JSONPtr(NewOperationResource(ctx,
			operation,
			NewAssetResource(ctx,
				asset, authentication.Get(ctx).User, c.mintHost))),
	})
}
