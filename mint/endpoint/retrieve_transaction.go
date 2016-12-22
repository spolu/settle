package endpoint

import (
	"context"
	"net/http"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint/model"
	"goji.io/pat"
)

const (
	// EndPtRetrieveTransaction creates a new assset.
	EndPtRetrieveTransaction EndPtName = "RetrieveTransaction"
)

func init() {
	registrar[EndPtRetrieveTransaction] = NewRetrieveTransaction
}

// RetrieveTransaction retrieves a transaction based on its id. It is not
// authenticated and is used to propagate transactions.
type RetrieveTransaction struct {
	ID    string
	Token string
	Owner string
}

// NewRetrieveTransaction constructs and initialiezes the endpoint.
func NewRetrieveTransaction(
	r *http.Request,
) (Endpoint, error) {
	return &RetrieveTransaction{}, nil

}

// Validate validates the input parameters.
func (e *RetrieveTransaction) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "transaction"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Token = *token
	e.Owner = *owner

	return nil
}

// Execute executes the endpoint.
func (e *RetrieveTransaction) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	transaction := txStore.Get(ctx, e.ID)
	if transaction == nil {
		tx, err := model.LoadCanonicalTransactionByOwnerToken(ctx,
			e.Owner, e.Token)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		} else if tx == nil {
			tx, err = model.LoadPropagatedTransactionByOwnerToken(ctx,
				e.Owner, e.Token)
			if err != nil {
				return nil, nil, errors.Trace(err) // 500
			} else if tx == nil {
				return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
					404, "transaction_not_found",
					"The transaction you are trying to retrieve does not "+
						"exist: %s.", e.ID,
				))
			}
		}
		transaction = tx
	}

	operations := txStore.GetOperations(ctx, e.ID)
	if operations == nil {
		ops, err := model.LoadCanonicalOperationsByTransaction(ctx, e.ID)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
		operations = ops
	}

	crossings := txStore.GetCrossings(ctx, e.ID)
	if crossings == nil {
		crs, err := model.LoadCanonicalCrossingsByTransaction(ctx, e.ID)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
		crossings = crs
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusOK), &svc.Resp{
		"transaction": format.JSONPtr(model.NewTransactionResource(ctx,
			transaction, operations, crossings)),
	}, nil
}
