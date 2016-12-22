package endpoint

import (
	"context"
	"fmt"
	"net/http"

	"goji.io/pat"

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
	// EndPtListAssetBalances creates a new assset.
	EndPtListAssetBalances EndPtName = "ListAssetBalances"
)

func init() {
	registrar[EndPtListAssetBalances] = NewListAssetBalances
}

// ListAssetBalances returns a list of balances.
type ListAssetBalances struct {
	ListEndpoint
	Owner string
	Asset mint.AssetResource
}

// NewListAssetBalances constructs and initialiezes the endpoint.
func NewListAssetBalances(
	r *http.Request,
) (Endpoint, error) {
	return &ListAssetBalances{
		ListEndpoint: ListEndpoint{},
	}, nil
}

// Validate validates the input parameters.
func (e *ListAssetBalances) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	e.Owner = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username, mint.GetHost(ctx))

	// Validate id.
	asset, err := ValidateAsset(ctx, pat.Param(r, "asset"))
	if err != nil {
		return errors.Trace(err)
	}
	e.Asset = *asset

	// Validate that the authenticated owner owns the asset.
	if e.Owner != e.Asset.Owner {
		return errors.Trace(errors.NewUserErrorf(nil,
			400, "not_authorized",
			"You can only retrieve asset balances for assets owned by the "+
				"account you are currently authenticated with: %s. The "+
				"requested asset is owned by: %s.",
			e.Owner, e.Asset.Owner,
		))
	}

	return e.ListEndpoint.Validate(r)
}

// Execute executes the endpoint.
func (e *ListAssetBalances) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	balances, err := model.LoadBalanceListByAsset(ctx,
		e.ListEndpoint.CreatedBefore,
		e.ListEndpoint.Limit,
		e.Asset.Name,
	)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	l := []mint.BalanceResource{}
	for _, b := range balances {
		b := b
		l = append(l, model.NewBalanceResource(ctx, &b))
	}

	return ptr.Int(http.StatusOK), &svc.Resp{
		"balances": format.JSONPtr(l),
	}, nil
}
