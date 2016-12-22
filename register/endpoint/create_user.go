// OWNER: stan

package endpoint

import (
	"context"
	"fmt"
	"net/http"
	"net/smtp"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/ptr"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/register"
	"github.com/spolu/settle/register/model"
)

const (
	// EndPtCreateUser creates a new offer.
	EndPtCreateUser EndPtName = "CreateUser"
)

func init() {
	registrar[EndPtCreateUser] = NewCreateUser
}

// CreateUser a new user by username and email and send its secret over eail.
type CreateUser struct {
	Username string
	Email    string
}

// NewCreateUser constructs and initialiezes the endpoint.
func NewCreateUser(
	r *http.Request,
) (Endpoint, error) {
	return &CreateUser{}, nil
}

// Validate validates the input parameters.
func (e *CreateUser) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate username.
	username, err := ValidateUsername(ctx, r.PostFormValue("username"))
	if err != nil {
		return errors.Trace(err) // 400
	}
	e.Username = *username

	// Validate email.
	email, err := ValidateEmail(ctx, r.PostFormValue("email"))
	if err != nil {
		return errors.Trace(err) // 400
	}
	e.Email = *email

	return nil
}

// Execute executes the endpoint.
func (e *CreateUser) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	ctx = db.Begin(ctx, "register")
	defer db.LoggedRollback(ctx)

	user, err := model.CreateUser(ctx,
		e.Username,
		e.Email,
	)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case model.ErrUniqueConstraintViolation:
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				400, "username_taken",
				"A user already exists with the same username: %s.",
				e.Username,
			))
		default:
			return nil, nil, errors.Trace(err) // 500
		}
	}

	if auth, host := register.GetSMTP(ctx); auth != nil {
		from := fmt.Sprintf("register-%s@%s",
			env.Get(ctx).Environment, register.GetMint(ctx))

		logging.Logf(ctx,
			"Sending email: username=%s email=%s",
			user.Username, user.Email)

		err := smtp.SendMail(
			host, *auth, from, []string{user.Email},
			[]byte("foo"))
		if err != nil {
			return nil, nil, errors.Trace(errors.NewUserErrorf(err,
				400, "email_failed",
				"The credentials email failed to be sent to: %s.",
				e.Email,
			))
		}
	}

	db.Commit(ctx)

	logging.Logf(ctx,
		"Created user: id=%s created=%q status=%s "+
			"username=%s email=%s",
		user.Token, user.Created, user.Status, user.Username, user.Email)

	return ptr.Int(http.StatusCreated), &svc.Resp{
		"user": format.JSONPtr(model.NewUserResource(ctx, user)),
	}, nil
}
