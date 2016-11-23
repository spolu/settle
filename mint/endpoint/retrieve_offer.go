package endpoint

import (
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
	// EndPtRetrieveOffer creates a new assset.
	EndPtRetrieveOffer EndPtName = "RetrieveOffer"
)

func init() {
	registrar[EndPtRetrieveOffer] = NewRetrieveOffer
}

// RetrieveOffer retrieves an offer based on its id. It is not authenticated
// and is used to verify offers when they get propagated.
type RetrieveOffer struct {
	ID    string
	Token string
	Owner string
}

// NewRetrieveOffer constructs and initialiezes the endpoint.
func NewRetrieveOffer(
	r *http.Request,
) (Endpoint, error) {
	return &RetrieveOffer{}, nil

}

// Validate validates the input parameters.
func (e *RetrieveOffer) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "offer"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Token = *token
	e.Owner = *owner

	return nil
}

// Execute executes the endpoint.
func (e *RetrieveOffer) Execute(
	r *http.Request,
) (*int, *svc.Resp, error) {
	ctx := r.Context()

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	offer, err := model.LoadCanonicalOfferByOwnerToken(ctx, e.Owner, e.Token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if offer == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "offer_not_found",
			"The offer you are trying to retrieve does not exist: %s.",
			e.ID,
		))
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"offer": format.JSONPtr(mint.NewOfferResource(ctx, offer)),
	}, nil
}
