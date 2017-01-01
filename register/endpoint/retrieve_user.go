// OWNER: stan

package endpoint

import (
	"context"
	"fmt"
	"log"
	"net/http"

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
	// EndPtRetrieveUser creates a new offer.
	EndPtRetrieveUser EndPtName = "RetrieveUser"
)

func init() {
	registrar[EndPtRetrieveUser] = NewRetrieveUser
}

// RetrieveUser a new user by username and email and send its secret over eail.
type RetrieveUser struct {
	Username string
	Secret   string
}

// NewRetrieveUser constructs and initialiezes the endpoint.
func NewRetrieveUser(
	r *http.Request,
) (Endpoint, error) {
	return &RetrieveUser{}, nil
}

// Validate validates the input parameters.
func (e *RetrieveUser) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate username.
	username, err := ValidateUsername(ctx, r.PostFormValue("username"))
	if err != nil {
		return errors.Trace(err) // 400
	}
	e.Username = *username

	// Validate secret.
	e.Secret = r.URL.Query().Get("secret")

	return nil
}

// Execute executes the endpoint.
func (e *RetrieveUser) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "register")
	defer db.LoggedRollback(ctx)

	user, err := model.LoadUserByUsername(ctx, e.Username)
	if err != nil {
		return nil, nil, errors.Trace(err) // 500
	} else if user == nil || user.Secret != e.Secret {
		return nil, nil, errors.Trace(errors.NewUserErrorf(err,
			400, "user_not_found",
			"The username and secret pair you specified is not associated "+
				"with any existing user.",
		))
	}

	if user.Status != register.UsrStVerified {
		user.Status = register.UsrStVerified
		err := user.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}
	}

	db.Commit(ctx)

	// If the user was not yet created on the mint, do so with two successive
	// transactions, one to create or update (in case there was an issue) the
	// user on the mint and the other to update the register user
	// representation.
	if user.MintToken == nil {
		ctx = db.Begin(ctx, "mint")
		defer db.LoggedRollback(ctx)

		u, err := mintmodel.LoadUserByUsername(ctx, user.Username)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}

		if u != nil {
			err := u.UpdatePassword(ctx, user.Password)
			if err != nil {
				return nil, nil, errors.Trace(err) // 500
			}
			err = u.Save(ctx)
			if err != nil {
				return nil, nil, errors.Trace(err) // 500
			}

			logging.Logf(ctx,
				"Updated mint user: id=%s created=%q username=%s",
				u.Token, u.Created, u.Username)
		} else {
			u, err = mintmodel.CreateUser(ctx, user.Username, user.Password)
			if err != nil {
				log.Fatal(errors.Details(err))
			}

			logging.Logf(ctx,
				"Created mint user: id=%s created=%q username=%s",
				u.Token, u.Created, u.Username)
		}

		db.Commit(ctx)

		ctx = db.Begin(ctx, "register")
		defer db.LoggedRollback(ctx)

		user.MintToken = &u.Token
		err := user.Save(ctx)
		if err != nil {
			return nil, nil, errors.Trace(err) // 500
		}

		db.Commit(ctx)
	}

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"user": format.JSONPtr(model.NewUserResource(ctx, user)),
		"credentials": format.JSONPtr(register.CredentialsResource{
			Address: fmt.Sprintf("%s@%s",
				user.Username, register.GetMint(ctx)),
			Password: user.Password,
		}),
	}, nil
}
