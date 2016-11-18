package endpoint

import (
	"fmt"
	"math/big"
	"net/http"

	"goji.io/pat"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
)

const (
	// EndPtCreateOperation creates a new operation.
	EndPtCreateOperation EndPtName = "CreateOperation"
)

func init() {
	registrar[EndPtCreateOperation] = NewCreateOperation
}

// CreateOperation creates a new operation that moves asset from a source
// balance to a destination balance:
// - no `source` specified: issuance.
// - no `destination` specified: annihilation.
// - both specified: transfer from a balance to another.
// Only the asset creator can create operation on an asset. To transfer money,
// users owning an asset whould use transactions.
type CreateOperation struct {
	Owner      string
	Asset      mint.AssetResource
	Amount     big.Int
	SrcAddress *string
	DstAddress *string
}

// NewCreateOperation constructs and initialiezes the endpoint.
func NewCreateOperation(
	r *http.Request,
) (Endpoint, error) {
	return &CreateOperation{}, nil
}

// Validate validates the input parameters.
func (e *CreateOperation) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	e.Owner = fmt.Sprintf("%s@%s",
		authentication.Get(ctx).User.Username,
		env.Get(ctx).Config[mint.EnvCfgMintHost])

	// Validate asset.
	a, err := mint.AssetResourceFromName(ctx, pat.Param(r, "asset"))
	if err != nil {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "asset_invalid",
			"The asset name you provided is invalid: %s.",
			pat.Param(r, "asset"),
		))
	}
	username, host, err := mint.UsernameAndMintHostFromAddress(ctx, a.Owner)
	if err != nil {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "asset_invalid",
			"The asset name you provided has an invalid issuer address: %s.",
			a.Owner,
		))
	}
	e.Asset = *a

	// Validate that the issuer is attempting to create the operation.
	if host != env.Get(ctx).Config[mint.EnvCfgMintHost] ||
		username != authentication.Get(ctx).User.Username {
		return errors.Trace(errors.NewUserErrorf(nil,
			400, "operation_not_authorized",
			"You can only create operations for assets created by the "+
				"account you are currently authenticated with: %s@%s. This "+
				"operation's asset was created by: %s@%s. If you own %s, "+
				"and wish to transfer some of it, you should create a "+
				"transaction directly from your mint instead.",
			authentication.Get(ctx).User.Username,
			env.Get(ctx).Config[mint.EnvCfgMintHost],
			username, host, a.Name,
		))
	}

	// Validate amount.
	var amount big.Int
	_, success := amount.SetString(r.PostFormValue("amount"), 10)
	if !success ||
		amount.Cmp(new(big.Int)) < 0 ||
		amount.Cmp(model.MaxAssetAmount) >= 0 {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "amount_invalid",
			"The amount you provided is invalid: %s. Amounts must be "+
				"integers between 0 and 2^128.",
			r.PostFormValue("amount"),
		))
	}
	e.Amount = amount

	// Validate source
	var srcAddress *string
	if r.PostFormValue("source") != "" {
		addr, err := mint.NormalizedAddress(ctx, r.PostFormValue("source"))
		if err != nil {
			return errors.Trace(errors.NewUserErrorf(err,
				400, "source_invalid",
				"The source address you provided is invalid: %s.",
				*srcAddress,
			))
		}
		srcAddress = &addr
	}
	e.SrcAddress = srcAddress

	// Validate destination
	var dstAddress *string
	if r.PostFormValue("destination") != "" {
		addr, err := mint.NormalizedAddress(ctx, r.PostFormValue("destination"))
		if err != nil {
			return errors.Trace(errors.NewUserErrorf(err,
				400, "destination_invalid",
				"The destination address you provided is invalid: %s.",
				*dstAddress,
			))
		}
		dstAddress = &addr
	}
	e.DstAddress = dstAddress

	if srcAddress == nil && dstAddress == nil {
		return errors.Trace(errors.NewUserErrorf(err,
			400, "operation_invalid",
			"The operation has no source and no destination. You must "+
				"specify at least one of them (no source: issuance; no "+
				"destination: annihilation; source and destination: "+
				"transfer).",
		))
	}

	return nil
}

// Execute executes the endpoint.
func (e *CreateOperation) Execute(
	r *http.Request,
) (*int, *svc.Resp, error) {
	ctx := r.Context()

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	asset, err := model.LoadAssetByOwnerCodeScale(ctx,
		e.Owner, e.Asset.Code, e.Asset.Scale)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if asset == nil {
		return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "asset_not_found",
			"The asset you are trying to operate does not exist: %s. Try "+
				"creating it first.",
			e.Asset.Name,
		))
	}
	assetName := fmt.Sprintf(
		"%s[%s.%d]",
		asset.Owner, asset.Code, asset.Scale)

	var srcBalance *model.Balance
	if e.SrcAddress != nil {
		srcBalance, err = model.LoadBalanceByAssetHolder(ctx,
			assetName, *e.SrcAddress)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		} else if srcBalance == nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(nil,
				400, "source_invalid",
				"The source address you provided has no existing balance: %s.",
				*e.SrcAddress,
			))
		}
	}

	var dstBalance *model.Balance
	if e.DstAddress != nil {
		dstBalance, err = model.LoadOrCreateBalanceByAssetHolder(ctx,
			authentication.Get(ctx).User.Token,
			e.Owner,
			assetName, *e.DstAddress)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	operation, err := model.CreateCanonicalOperation(ctx,
		authentication.Get(ctx).User.Token,
		e.Owner,
		assetName, e.SrcAddress, e.DstAddress, model.Amount(e.Amount))
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	if dstBalance != nil {
		(*big.Int)(&dstBalance.Value).Add(
			(*big.Int)(&dstBalance.Value), (*big.Int)(&operation.Amount))

		// Checks if the dstBalance is positive and not overflown.
		b := (*big.Int)(&dstBalance.Value)
		if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
			b.Cmp(new(big.Int)) < 0 {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				400, "amount_invalid",
				"The resulting destination balance is invalid: %s. The "+
					"balance must be an integer between 0 and 2^128.",
				b.String(),
			))
		}

		err = dstBalance.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	if srcBalance != nil {
		(*big.Int)(&srcBalance.Value).Sub(
			(*big.Int)(&srcBalance.Value), (*big.Int)(&operation.Amount))

		// Checks if the srcBalance is positive and not overflown.
		b := (*big.Int)(&srcBalance.Value)
		if new(big.Int).Abs(b).Cmp(model.MaxAssetAmount) >= 0 ||
			b.Cmp(new(big.Int)) < 0 {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				400, "amount_invalid",
				"The resulting source balance is invalid: %s. The "+
					"balance must be an integer between 0 and 2^128.",
				b.String(),
			))
		}

		err = srcBalance.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	db.Commit(ctx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"operation": format.JSONPtr(mint.NewOperationResource(ctx,
			operation, asset)),
	}, nil
}
