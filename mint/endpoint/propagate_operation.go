package endpoint

import (
	"context"
	"math/big"
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
	// EndPtPropagateOperation creates a new operation.
	EndPtPropagateOperation EndPtName = "PropagateOperation"
)

func init() {
	registrar[EndPtPropagateOperation] = NewPropagateOperation
}

// PropagateOperation retrieves a canonical operation and creates a local
// propagated copy of it.
type PropagateOperation struct {
	Client *mint.Client

	ID    string
	Owner string
	Token string
}

// NewPropagateOperation constructs and initialiezes the endpoint.
func NewPropagateOperation(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}

	return &PropagateOperation{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *PropagateOperation) Validate(
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
func (e *PropagateOperation) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	// Technically this is a propagation so out of consistency with other
	// endpoints we call ExecutePropagated.
	return e.ExecutePropagated(ctx)
}

// ExecutePropagated executes the propagation of an operation (involved mint).
func (e *PropagateOperation) ExecutePropagated(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	operation, err := e.Client.RetrieveOperation(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "propagation_failed",
			"Failed to retrieve canonical operation: %s", e.ID,
		))
	}

	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, operation.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid operation id: %s", operation.ID,
		))
	}

	if e.ID != operation.ID {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected operation id: %s expected %s", operation.ID, e.ID,
		))
	}
	if e.Owner != owner || operation.Owner != owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected operation owner: %s expected %s", owner, e.Owner,
		))
	}
	if e.Token != token {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected operation token: %s expected %s", token, e.Token,
		))
	}
	if operation.Status != mint.TxStSettled {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Operation status is %s. Only settled operations "+
				"can be propagated.", operation.Status,
		))
	}

	asset, err := mint.AssetResourceFromName(ctx, operation.Asset)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid operation asset: %s", operation.Asset,
		))
	}
	if asset.Owner != operation.Owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Operation and asset owner mismatch: %s expected %s",
			operation.Asset, operation.Owner,
		))
	}

	srcUser, srcMint, err := mint.UsernameAndMintHostFromAddress(ctx,
		operation.Source)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid operation source: %s", operation.Source,
		))
	}
	dstUser, dstMint, err := mint.UsernameAndMintHostFromAddress(ctx,
		operation.Destination)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
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
			402, "propagation_failed",
			"Received operation with no impact on any of this mint users.",
		))
	}

	u, err := model.LoadUserByUsername(ctx, user)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if u == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"User impacted by operation does not exist: %s@%s",
			user, mint.GetHost(ctx),
		))
	}

	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	code := http.StatusCreated

	op, err := model.LoadPropagatedOperationByOwnerToken(ctx, owner, token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if op != nil {
		// Nothing to do: an operation is immutable once settled.
		code = http.StatusOK
	} else {
		// Create propagated operation locally.
		op, err = model.CreatePropagatedOperation(ctx,
			owner,
			token,
			time.Unix(0, operation.Created*mint.TimeResolutionNs),
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

		mint.Logf(ctx,
			"Propagated operation: id=%s[%s] created=%q propagation=%s "+
				"asset=%s source=%s destination=%s amount=%s "+
				"status=%s transaction=%s",
			op.Owner, op.Token, op.Created, op.Propagation, op.Asset,
			op.Source, op.Destination, (*big.Int)(&op.Amount).String(),
			op.Status, *op.Transaction)
	}

	db.Commit(ctx)

	return ptr.Int(code), &svc.Resp{
		"operation": format.JSONPtr(model.NewOperationResource(ctx, op)),
	}, nil
}
