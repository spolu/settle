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
	// EndPtRetrieveOperation creates a new assset.
	EndPtRetrieveOperation EndPtName = "RetrieveOperation"
)

func init() {
	registrar[EndPtRetrieveOperation] = NewRetrieveOperation
}

// RetrieveOperation retrieves an operation based on its id. It is not
// authenticated and is used to verify operations when they get propagated.
type RetrieveOperation struct {
	ID    string
	Token string
	Owner string
}

// NewRetrieveOperation constructs and initialiezes the endpoint.
func NewRetrieveOperation(
	r *http.Request,
) (Endpoint, error) {
	return &RetrieveOperation{}, nil

}

// Validate validates the input parameters.
func (e *RetrieveOperation) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "operation"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Token = *token
	e.Owner = *owner

	return nil
}

// Execute executes the endpoint.
func (e *RetrieveOperation) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	operation, err := model.LoadCanonicalOperationByOwnerToken(ctx,
		e.Owner, e.Token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if operation == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			404, "operation_not_found",
			"The operation you are trying to retrieve does not exist: %s.",
			e.ID,
		))
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"operation": format.JSONPtr(model.NewOperationResource(ctx, operation)),
	}, nil
}
