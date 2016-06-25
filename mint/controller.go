package mint

import (
	"fmt"
	"net/http"
	"strconv"

	"goji.io/pat"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/respond"
	"github.com/spolu/settle/lib/svc"
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

	respond.Success(ctx, w, svc.Resp{
		"asset": format.JSONPtr(AssetResource{
			ID:       asset.Token,
			Created:  asset.Created.UnixNano() / (1000 * 1000),
			Livemode: asset.Livemode,
			Name: fmt.Sprintf(
				"%s@%s:%s.%d",
				authentication.Get(ctx).User.Username, c.mintHost,
				asset.Code, asset.Scale,
			),
			Issuer: fmt.Sprintf(
				"%s@%s",
				authentication.Get(ctx).User.Username, c.mintHost,
			),
			Code:  asset.Code,
			Scale: asset.Scale,
		}),
	})
}

// IssueAsset controls the issuance of an asset.
func (c *controller) IssueAsset(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	a, err := c.client.AssetResourceFromName(
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

	username, host, err := c.client.UsernameAndMintHostFromAddress(
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

	respond.Success(ctx, w, svc.Resp{
		"asset": format.JSONPtr(a),
	})
}
