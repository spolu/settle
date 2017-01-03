// OWNER: stan

package endpoint

import (
	"context"
	"fmt"
	"net/http"

	"goji.io/pat"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	mintmodel "github.com/spolu/settle/mint/model"
	"github.com/spolu/settle/register"
	"github.com/spolu/settle/register/model"
)

const (
	// EndPtRollUser rolls a user password.
	EndPtRollUser EndPtName = "RollUser"
)

func init() {
	registrar[EndPtRollUser] = NewRollUser
}

// RollUser a new user by username and email and send its secret over eail.
type RollUser struct {
	Username string
	Secret   string
}

// NewRollUser constructs and initialiezes the endpoint.
func NewRollUser(
	r *http.Request,
) (Endpoint, error) {
	return &RollUser{}, nil
}

// Validate validates the input parameters.
func (e *RollUser) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate username.
	username, err := ValidateUsername(ctx, pat.Param(r, "username"))
	if err != nil {
		return errors.Trace(err) // 400
	}
	e.Username = *username

	// Validate secret.
	e.Secret = r.PostFormValue("secret")

	return nil
}

// Execute executes the endpoint.
func (e *RollUser) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	regCtx := db.Begin(ctx, "register")
	defer db.LoggedRollback(regCtx)

	user, err := model.LoadUserByUsername(regCtx, e.Username)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if user == nil || user.Secret != e.Secret {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			400, "user_not_found",
			"The username and secret pair you specified is not associated "+
				"with any existing user.",
		))
	}

	err = user.RollPassword(ctx)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}
	err = user.Save(regCtx)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	}

	if user.MintToken != nil {
		mintCtx := db.Begin(ctx, "mint")
		defer db.LoggedRollback(mintCtx)

		u, err := mintmodel.LoadUserByUsername(mintCtx, user.Username)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		} else if u == nil {
			return nil, nil, errors.Trace(
				errors.Newf("Mint user not found: %s", user.Username)) // 500
		}

		err = u.UpdatePassword(mintCtx, user.Password)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
		err = u.Save(mintCtx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}

		logging.Logf(mintCtx,
			"Updated mint user: id=%s created=%q username=%s",
			u.Token, u.Created, u.Username)

		db.Commit(mintCtx)
	}

	logging.Logf(regCtx,
		"Rolled user: id=%s created=%q username=%s status=%s",
		user.Token, user.Created, user.Username, user.Status)

	db.Commit(regCtx)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"user": format.JSONPtr(model.NewUserResource(ctx, user)),
		"credentials": format.JSONPtr(register.CredentialsResource{
			Address: fmt.Sprintf("%s@%s",
				user.Username, register.GetMint(ctx)),
			Password: user.Password,
		}),
	}, nil
}
