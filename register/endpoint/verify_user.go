// OWNER: stan

package endpoint

import (
	"context"
	"net/http"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/svc"
)

const (
	// EndPtVerifyUser creates a new offer.
	EndPtVerifyUser EndPtName = "VerifyUser"
)

func init() {
	registrar[EndPtVerifyUser] = NewVerifyUser
}

// VerifyUser a new user by username and email and send its secret over eail.
type VerifyUser struct {
	Username          string
	Secret            string
	IP                string
	ReCAPTCHAResponse string
}

// NewVerifyUser constructs and initialiezes the endpoint.
func NewVerifyUser(
	r *http.Request,
) (Endpoint, error) {
	return &VerifyUser{}, nil
}

// Validate validates the input parameters.
func (e *VerifyUser) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate username.
	username, err := ValidateUsername(ctx, r.PostFormValue("username"))
	if err != nil {
		return errors.Trace(err) // 400
	}
	e.Username = *username

	// No need to validate these values as they are going to be validated by
	// their usage at execution.
	e.Secret = r.PostFormValue("secret")
	e.IP = r.PostFormValue("ip")
	e.ReCAPTCHAResponse = r.PostFormValue("recaptcha_response")

	return nil
}

// Execute executes the endpoint.
func (e *VerifyUser) Execute(
	ctx context.Context,
) (*int, *svc.Resp, error) {
	return nil, nil, nil
}
