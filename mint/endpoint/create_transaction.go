// OWNER: stan

package endpoint

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/authentication"
)

const (
	// EndPtCreateTransaction creates a new transaction.
	EndPtCreateTransaction EndPtName = "CreateTransaction"
)

func init() {
	registrar[EndPtCreateTransaction] = NewCreateTransaction
}

// CreateTransaction creates a new transaction.
type CreateTransaction struct {
	Client *mint.Client

	ID         string               // if propagated
	Token      string               // if propagated
	Owner      string               // both
	Pair       []mint.AssetResource // if canonical
	BasePrice  big.Int              // if canonical
	QuotePrice big.Int              // if canonical
	Amount     big.Int              // if canonical
	Path       []string             // if canonical
}

// NewCreateTransaction constructs and initialiezes the endpoint.
func NewCreateTransaction(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}
	return &CreateTransaction{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *CreateTransaction) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	if authentication.Get(ctx).Status != authentication.AutStSucceeded {
		// Validate id.
		id, owner, token, err := ValidateID(r, r.PostFormValue("id"))
		if err != nil {
			return errors.Trace(err)
		}
		e.ID = *id
		e.Token = *token
		e.Owner = *owner

		return nil
	}

	e.Owner = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		env.Get(ctx).Config[mint.EnvCfgMintHost])

	// Validate asset pair.
	pair, err := ValidateAssetPair(r, r.PostFormValue("pair"))
	if err != nil {
		return errors.Trace(err) // 400
	}
	e.Pair = pair

	// Validate price.
	basePrice, quotePrice, err := ValidatePrice(r, r.PostFormValue("price"))
	if err != nil {
		return errors.Trace(err)
	}
	e.BasePrice = *basePrice
	e.QuotePrice = *quotePrice

	// Validate amount.
	amount, err := ValidateAmount(r, r.PostFormValue("amount"))
	if err != nil {
		return errors.Trace(err)
	}
	e.Amount = *amount

	// Validate path.
	if r.PostForm == nil {
		err := r.ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			return errors.Trace(err) // 500
		}
	}
	path, err := ValidatePath(r, r.PostForm["path[]"])
	if err != nil {
		return errors.Trace(err)
	}
	e.Path = path

	return nil
}

// Execute executes the endpoint.
func (e *CreateTransaction) Execute(
	r *http.Request,
) (*int, *svc.Resp, error) {
	ctx := r.Context()

	if authentication.Get(ctx).Status == authentication.AutStSucceeded {
		return e.ExecuteCanonical(r)
	}
	return e.ExecutePropagated(r)
}

// ExecuteCanonical executes the creation of a canonical transaction (owner
// mint).
func (e *CreateTransaction) ExecuteCanonical(
	r *http.Request,
) (*int, *svc.Resp, error) {
	//ctx := r.Context()

	// TODO(stan): create offer, store in memory cache

	return ptr.Int(http.StatusCreated), &svc.Resp{}, nil
}

// ExecutePropagated executes the creation of a propagated transaction
// (involved mint).
func (e *CreateTransaction) ExecutePropagated(
	r *http.Request,
) (*int, *svc.Resp, error) {
	//ctx := r.Context()

	// TODO(stan): retrieve transaction from ID
	// TODO(stan): forward API call

	return ptr.Int(http.StatusCreated), &svc.Resp{}, nil
}
