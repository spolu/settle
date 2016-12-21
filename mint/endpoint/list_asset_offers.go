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
	// EndPtListAssetOffers creates a new assset.
	EndPtListAssetOffers EndPtName = "ListAssetOffers"
)

func init() {
	registrar[EndPtListAssetOffers] = NewListAssetOffers
}

// ListAssetOffers returns a list of offers.
type ListAssetOffers struct {
	ListEndpoint
	Asset       mint.AssetResource
	Propagation mint.PgType
}

// NewListAssetOffers constructs and initialiezes the endpoint.
func NewListAssetOffers(
	r *http.Request,
) (Endpoint, error) {
	return &ListAssetOffers{
		ListEndpoint: ListEndpoint{},
	}, nil
}

// Validate validates the input parameters.
func (e *ListAssetOffers) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	asset, err := ValidateAsset(ctx, pat.Param(r, "asset"))
	if err != nil {
		return errors.Trace(err)
	}
	e.Asset = *asset

	// Validate propagation.
	propagation, err := ValidatePropagation(ctx,
		r.URL.Query().Get("propagation"))
	if err != nil {
		return errors.Trace(err)
	}
	e.Propagation = *propagation

	return e.ListEndpoint.Validate(r)
}

// Execute executes the endpoint.
func (e *ListAssetOffers) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	var offers []model.Offer
	var err error

	switch e.Propagation {
	case mint.PgTpCanonical:
		offers, err = model.LoadOfferListByBaseAsset(ctx,
			e.ListEndpoint.CreatedBefore,
			e.ListEndpoint.Limit,
			e.Asset.Name,
		)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	case mint.PgTpPropagated:
		offers, err = model.LoadOfferListByQuoteAsset(ctx,
			e.ListEndpoint.CreatedBefore,
			e.ListEndpoint.Limit,
			e.Asset.Name,
		)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	db.Commit(ctx)

	l := []mint.OfferResource{}
	for _, o := range offers {
		o := o
		l = append(l, model.NewOfferResource(ctx, &o))
	}

	return ptr.Int(http.StatusOK), &svc.Resp{
		"offers": format.JSONPtr(l),
	}, nil
}
