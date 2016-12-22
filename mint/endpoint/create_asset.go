package endpoint

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
)

const (
	// EndPtCreateAsset creates a new assset.
	EndPtCreateAsset EndPtName = "CreateAsset"
)

func init() {
	registrar[EndPtCreateAsset] = NewCreateAsset
}

// CreateAsset controls the creation of new assets.
type CreateAsset struct {
	Owner string
	Code  string
	Scale int8
}

// NewCreateAsset constructs and initialiezes the endpoint.
func NewCreateAsset(
	r *http.Request,
) (Endpoint, error) {
	return &CreateAsset{}, nil
}

// Validate validates the input parameters.
func (e *CreateAsset) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	e.Owner = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username, mint.GetHost(ctx))

	code := r.PostFormValue("code")
	if !model.AssetCodeRegexp.MatchString(code) {
		return errors.Trace(errors.NewUserErrorf(nil,
			400, "code_invalid",
			"The asset code provided is invalid: %s. Asset codes can use "+
				"alphanumeric upercased and `-` characters only.",
			code,
		))
	}
	e.Code = code

	scale, err := strconv.ParseInt(r.PostFormValue("scale"), 10, 8)
	if err != nil ||
		(int8(scale) < model.AssetMinScale ||
			int8(scale) > model.AssetMaxScale) {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "scale_invalid",
			"The asset scale provided is invalid: %s. Asset scales must be "+
				"integers between %d and %d.",
			r.PostFormValue("scale"), model.AssetMinScale, model.AssetMaxScale,
		))
	}
	e.Scale = int8(scale)

	return nil
}

// Execute executes the endpoint.
func (e *CreateAsset) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	asset, err := model.CreateCanonicalAsset(ctx,
		e.Owner,
		e.Code,
		e.Scale,
	)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case model.ErrUniqueConstraintViolation:
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				400, "asset_already_exists",
				"You already created an asset with the same code and "+
					"scale: %s.%d.",
				e.Code, e.Scale,
			))
		default:
			return nil, nil, errors.Trace(err) // 500
		}
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"asset": format.JSONPtr(model.NewAssetResource(ctx, asset)),
	}, nil
}
