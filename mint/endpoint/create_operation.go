// OWNER: stan

package endpoint

import (
	"context"
	"net/http"
	"time"

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
	// EndPtCreateOperation creates a new operation.
	EndPtCreateOperation EndPtName = "CreateOperation"
)

func init() {
	registrar[EndPtCreateOperation] = NewCreateOperation
}

// CreateOperation is used for the propagation of operations across mints. It's
// an unauthenticated endpoint called by canonical mints (or clients if
// necessary) that triggers the retrieval and local storage of an operation.
// Note that canonical Operations are exclusively created by transactions.
type CreateOperation struct {
	Client *mint.Client

	ID    string
	Owner string
	Token string
}

// NewCreateOperation constructs and initialiezes the endpoint.
func NewCreateOperation(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}

	return &CreateOperation{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *CreateOperation) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "operation"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Owner = *owner
	e.Token = *token

	return nil
}

// Execute executes the endpoint.
func (e *CreateOperation) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	// Technically this is a propagation so out of consistency with other
	// endpoints we call ExecutePropagated.
	return e.ExecutePropagated(ctx)
}

// ExecutePropagated executes the settlement of a propagated transaction
// (involved mint).
func (e *CreateOperation) ExecutePropagated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	operation, err := e.Client.RetrieveOperation(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "propogation_failed",
			"Failed to retrieve canonical operation: %s", e.ID,
		))
	}

	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, operation.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Received invalid operation id: %s", operation.ID,
		))
	}

	if e.ID != operation.ID {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Unexpected operation id: %s expected %s", operation.ID, e.ID,
		))
	}
	if e.Owner != owner || operation.Owner != owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Unexpected operation owner: %s expected %s", owner, e.Owner,
		))
	}
	if e.Token != token {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Unexpected operation token: %s expected %s", token, e.Token,
		))
	}
	if operation.Status != mint.TxStSettled {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Operation is not settled: status is %s. Only settled operations "+
				"can be propagated.", operation.Status,
		))
	}

	asset, err := mint.AssetResourceFromName(ctx, operation.Asset)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Received invalid operation asset: %s", operation.Asset,
		))
	}
	if asset.Owner != operation.Owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Operation and asset owner mismatch: %s expected %s",
			operation.Asset, operation.Owner,
		))
	}

	srcUser, srcMint, err := mint.UsernameAndMintHostFromAddress(ctx,
		operation.Source)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Received invalid operation source: %s", operation.Source,
		))
	}
	dstUser, dstMint, err := mint.UsernameAndMintHostFromAddress(ctx,
		operation.Destination)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Received invalid operation destination: %s", operation.Destination,
		))
	}

	user := ""
	if srcMint == mint.GetHost(ctx) {
		user = srcUser
	} else if dstMint == mint.GetHost(ctx) {
		user = dstUser
	} else {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"Received operation with no impact on any of this mint users.",
		))
	}

	u, err := model.LoadUserByUsername(ctx, user)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if u == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propogation_failed",
			"User impacted by operation does not exist: %s@%s",
			user, mint.GetHost(ctx),
		))
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	// Create propagated operation locally.
	op, err := model.CreatePropagatedOperation(ctx,
		token,
		time.Unix(0, operation.Created*mint.TimeResolutionNs),
		owner,
		operation.Asset,
		operation.Source,
		operation.Destination,
		model.Amount(*operation.Amount),
		operation.Status,
		operation.Transaction,
		operation.TransactionHop,
	)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"operation": format.JSONPtr(model.NewOperationResource(ctx, op)),
	}, nil
}
