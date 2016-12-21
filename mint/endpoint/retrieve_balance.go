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
	"github.com/spolu/settle/mint/model"
)

const (
	// EndPtRetrieveBalance creates a new assset.
	EndPtRetrieveBalance EndPtName = "RetrieveBalance"
)

func init() {
	registrar[EndPtRetrieveBalance] = NewRetrieveBalance
}

// RetrieveBalance retrieves an balance based on its id. It is not
// authenticated and is used to verify balances when they get propagated.
type RetrieveBalance struct {
	ID    string
	Token string
	Owner string
}

// NewRetrieveBalance constructs and initialiezes the endpoint.
func NewRetrieveBalance(
	r *http.Request,
) (Endpoint, error) {
	return &RetrieveBalance{}, nil

}

// Validate validates the input parameters.
func (e *RetrieveBalance) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "balance"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Token = *token
	e.Owner = *owner

	return nil
}

// Execute executes the endpoint.
func (e *RetrieveBalance) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	balance, err := model.LoadCanonicalBalanceByOwnerToken(ctx,
		e.Owner, e.Token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if balance == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "balance_not_found",
			"The balance you are trying to retrieve does not exist: %s.",
			e.ID,
		))
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"balance": format.JSONPtr(model.NewBalanceResource(ctx, balance)),
	}, nil
}
