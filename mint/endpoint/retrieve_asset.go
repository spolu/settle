package endpoint

import (
	"context"
	"net/http"

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
	// EndPtRetrieveAsset creates a new assset.
	EndPtRetrieveAsset EndPtName = "RetrieveAsset"
)

func init() {
	registrar[EndPtRetrieveAsset] = NewRetrieveAsset
}

// RetrieveAsset retrieves an asset based on its name. It is not authenticated
// and is used to verify the existence of an asset.
type RetrieveAsset struct {
	Asset mint.AssetResource
}

// NewRetrieveAsset constructs and initialiezes the endpoint.
func NewRetrieveAsset(
	r *http.Request,
) (Endpoint, error) {
	return &RetrieveAsset{}, nil

}

// Validate validates the input parameters.
func (e *RetrieveAsset) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	asset, err := ValidateAsset(ctx, pat.Param(r, "asset"))
	if err != nil {
		return errors.Trace(err)
	}
	e.Asset = *asset

	return nil
}

// Execute executes the endpoint.
func (e *RetrieveAsset) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	asset, err := model.LoadCanonicalAssetByOwnerCodeScale(ctx,
		e.Asset.Owner, e.Asset.Code, e.Asset.Scale)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if asset == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "asset_not_found",
			"The asset you are trying to retrieve does not exist: %s.",
			e.Asset.Name,
		))
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"asset": format.JSONPtr(model.NewAssetResource(ctx, asset)),
	}, nil
}
