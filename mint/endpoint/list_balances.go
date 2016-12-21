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
	// EndPtListBalances creates a new assset.
	EndPtListBalances EndPtName = "ListBalances"
)

func init() {
	registrar[EndPtListBalances] = NewListBalances
}

// ListBalances returns a list of balances.
type ListBalances struct {
	ListEndpoint
	Holder string
}

// NewListBalances constructs and initialiezes the endpoint.
func NewListBalances(
	r *http.Request,
) (Endpoint, error) {
	return &ListBalances{
		ListEndpoint: ListEndpoint{},
	}, nil
}

// Validate validates the input parameters.
func (e *ListBalances) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	e.Holder = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username, mint.GetHost(ctx))

	return e.ListEndpoint.Validate(r)
}

// Execute executes the endpoint.
func (e *ListBalances) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	balances, err := model.LoadBalanceListByHolder(ctx,
		e.ListEndpoint.CreatedBefore,
		e.ListEndpoint.Limit,
		e.Holder,
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
