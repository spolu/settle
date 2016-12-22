package endpoint

import (
	"context"
	"fmt"
	"net/http"

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
	// EndPtListAssets creates a new assset.
	EndPtListAssets EndPtName = "ListAssets"
)

func init() {
	registrar[EndPtListAssets] = NewListAssets
}

// ListAssets returns a list of assets.
type ListAssets struct {
	ListEndpoint
	Owner string
}

// NewListAssets constructs and initialiezes the endpoint.
func NewListAssets(
	r *http.Request,
) (Endpoint, error) {
	return &ListAssets{
		ListEndpoint: ListEndpoint{},
	}, nil
}

// Validate validates the input parameters.
func (e *ListAssets) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	e.Owner = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username, mint.GetHost(ctx))

	return e.ListEndpoint.Validate(r)
}

// Execute executes the endpoint.
func (e *ListAssets) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	assets, err := model.LoadAssetListByOwner(ctx,
		e.ListEndpoint.CreatedBefore,
		e.ListEndpoint.Limit,
		e.Owner,
	)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	l := []mint.AssetResource{}
	for _, a := range assets {
		a := a
		l = append(l, model.NewAssetResource(ctx, &a))
	}

	return ptr.Int(http.StatusOK), &svc.Resp{
		"assets": format.JSONPtr(l),
	}, nil
}
