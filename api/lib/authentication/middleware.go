package authentication

import (
	"net/http"

	"github.com/spolu/settl/lib/errors"
	"github.com/spolu/settl/lib/logging"
	"github.com/spolu/settl/lib/respond"
	"github.com/spolu/settl/model"

	"goji.io"

	"golang.org/x/net/context"
)

const (
	// statusKey the context.Context key to store the authentication status.
	statusKey string = "authentication.status"
)

// AutStatus indicates the status of the authentication.
type AutStatus string

const (
	// AutStSucceeded indicates a successful authentication.
	AutStSucceeded AutStatus = "succeeded"
	// AutStSkipped indicates a skipped authentication.
	AutStSkipped AutStatus = "skipped"
	// AutStFailed indicates a failed authentication.
	AutStFailed AutStatus = "failed"
)

// Status stores the authentication information.
type Status struct {
	Status  AutStatus
	Address string
}

// With stores the authentication information in a new context.
func With(
	ctx context.Context,
	status Status,
) context.Context {
	return context.WithValue(ctx, statusKey, status)
}

// Get retrieves the authenticaiton information form the context.
func Get(
	ctx context.Context,
) Status {
	return ctx.Value(statusKey).(Status)
}

// SkipList is the list of endpoints that do not require authentication.
var SkipList = []string{
	"/challenges",
}

type middleware struct {
	goji.Handler
}

// ServeHTTPC handles incoming HTTP requests and attempt to authenticate them.
func (m middleware) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	withStatus := With(ctx, Status{AutStFailed, ""})

	address, signature, _ := r.BasicAuth()
	challenge := r.Header.Get("Authorization-Challenge")
	skip := false
	for _, p := range SkipList {
		if r.URL.EscapedPath() == p {
			skip = true
		}
	}

	if skip {
		withStatus = With(ctx, Status{AutStSkipped, ""})
		logging.Logf(ctx, "Authentication: status=%q", Get(withStatus).Status)

		m.Handler.ServeHTTPC(withStatus, w, r)
		return
	}

	// Helper closure to log and return an authentication error.
	failedAuth := func(err error) {
		withStatus = With(ctx, Status{AutStFailed, ""})
		logging.Logf(ctx, "Authentication: status=%q", Get(withStatus).Status)

		respond.Error(withStatus, w, errors.Trace(err))
	}

	// Check that the challenge is valid.
	err := CheckChallenge(ctx, challenge, RootLiveKeypair)
	if err != nil {
		failedAuth(errors.Trace(err))
		return
	}

	// Verify the challenge signature passed as basic auth.
	err = VerifyChallenge(ctx, challenge, address, signature)
	if err != nil {
		failedAuth(errors.Trace(err))
		return
	}

	// Check that the challenge was never used.
	auth, err := model.LoadAuthenticationByChallenge(ctx, challenge)
	if err != nil {
		failedAuth(errors.Trace(err))
		return
	} else if auth != nil {
		failedAuth(errors.NewUserError(err,
			400, "challenge_already_used",
			"The challenge you provided was already used. You must "+
				"resolve a new challenge for each API request.",
		))
		return
	}

	auth, err = model.CreateAuthentication(ctx,
		r.Method, r.URL.String(), challenge, address, signature)
	if err != nil {
		failedAuth(errors.Trace(err))
		return
	}

	withStatus = With(ctx, Status{AutStSucceeded, address})
	logging.Logf(ctx,
		"Authentication: status=%q challenge=%q address=%q signature=%q",
		Get(withStatus).Status, challenge, address, signature)

	m.Handler.ServeHTTPC(withStatus, w, r)
}

// Middleware that authenticates API requests.
func Middleware(h goji.Handler) goji.Handler {
	return middleware{h}
}
