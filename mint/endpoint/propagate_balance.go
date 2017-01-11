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
	// EndPtPropagateBalance creates a new balance.
	EndPtPropagateBalance EndPtName = "PropagateBalance"
)

func init() {
	registrar[EndPtPropagateBalance] = NewPropagateBalance
}

// PropagateBalance fetches the balance propagated and creates a local
// propagated copy of it.
type PropagateBalance struct {
	Client *mint.Client

	ID    string
	Token string
	Owner string
}

// NewPropagateBalance constructs and initialiezes the endpoint.
func NewPropagateBalance(
	r *http.Request,
) (Endpoint, error) {
	ctx := r.Context()

	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return nil, errors.Trace(err) // 500
	}
	return &PropagateBalance{
		Client: client,
	}, nil
}

// Validate validates the input parameters.
func (e *PropagateBalance) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate id.
	id, owner, token, err := ValidateID(ctx, pat.Param(r, "balance"))
	if err != nil {
		return errors.Trace(err)
	}
	e.ID = *id
	e.Owner = *owner
	e.Token = *token

	return nil
}

// Execute executes the endpoint.
func (e *PropagateBalance) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	balance, err := e.Client.RetrieveBalance(ctx, e.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			402, "propagation_failed",
			"Failed to retrieve canonical balance: %s", e.ID,
		))
	}

	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, balance.ID)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid balance id: %s", balance.ID,
		))
	}

	if e.ID != balance.ID {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected balance id: %s expected %s", balance.ID, e.ID,
		))
	}
	if e.Owner != owner || balance.Owner != owner {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected balance owner: %s expected %s", owner, e.Owner,
		))
	}
	if e.Token != token {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Unexpected balance token: %s expected %s", token, e.Token,
		))
	}

	asset, err := mint.AssetResourceFromName(ctx, balance.Asset)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid balance asset: %s", balance.Asset,
		))
	}

	value, err := ValidateAmount(ctx, balance.Value.String())
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid balance value: %s", balance.Value.String(),
		))
	}

	user, host, err := mint.UsernameAndMintHostFromAddress(ctx, balance.Holder)
	if err != nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received invalid balance holder: %s", balance.Holder,
		))
	}
	if host != mint.GetHost(ctx) {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"Received balance whose holder is not one of this mint users.",
		))
	}

	u, err := model.LoadUserByUsername(ctx, user)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if u == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			402, "propagation_failed",
			"User impacted by balance does not exist: %s@%s",
			user, mint.GetHost(ctx),
		))
	}

	ctx = db.Begin(ctx, "mint")
	defer db.LoggedRollback(ctx)

	code := http.StatusCreated

	bal, err := model.LoadPropagatedBalanceByOwnerToken(ctx, owner, token)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if bal != nil {
		// Only the balance value is mutable.
		bal.Value = model.Amount(*value)

		err := bal.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
		code = http.StatusOK
	} else {
		// Create propagated balance locally.
		bal, err = model.CreatePropagatedBalance(ctx,
			owner,
			token,
			time.Unix(0, balance.Created*mint.TimeResolutionNs),
			asset.Name,
			balance.Holder,
			model.Amount(*value),
		)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}

		mint.Logf(ctx,
			"Propagated balance: id=%s[%s] created=%q propagation=%s "+
				"asset=%s holder=%s value=%s",
			bal.Owner, bal.Token, bal.Created, bal.Propagation, bal.Asset,
			bal.Holder, (*big.Int)(&bal.Value).String())
	}

	db.Commit(ctx)

	return ptr.Int(code), &svc.Resp{
		"balance": format.JSONPtr(model.NewBalanceResource(ctx, bal)),
	}, nil
}
