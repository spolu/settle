package authentication

import (
	"context"
	"net/http"
	"regexp"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/respond"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
)

// ContextKey is the type of the key used with context to carry contextual
// authentication status.
type ContextKey string

const (
	// statusKey the context.Context key to store the authentication status.
	statusKey ContextKey = "authentication.status"
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

// Status stores the authentication information, the status and authenticated
// user if applicable.
type Status struct {
	Status AutStatus
	User   *model.User
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

type middleware struct {
	http.Handler
}

// SkipRule defines a skip rule for authentication
type SkipRule struct {
	Method  string
	Pattern *regexp.Regexp
}

// SkipList is the list of endpoints that do not require authentication.
var SkipList = []*SkipRule{
	&SkipRule{"GET", regexp.MustCompile("^/offers/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},
	&SkipRule{"GET", regexp.MustCompile("^/operations/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},
	&SkipRule{"GET", regexp.MustCompile("^/transactions/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},
	&SkipRule{"GET", regexp.MustCompile("^/balances/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},

	&SkipRule{"POST", regexp.MustCompile("^/offers/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},
	&SkipRule{"POST", regexp.MustCompile("^/operations/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},
	&SkipRule{"POST", regexp.MustCompile("^/balances/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},

	&SkipRule{"POST", regexp.MustCompile("^/transactions/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},
	&SkipRule{"POST", regexp.MustCompile("^/transactions/[a-zA-Z0-9_\\+:@\\.\\[\\]]+/settle$")},
	&SkipRule{"POST", regexp.MustCompile("^/transactions/[a-zA-Z0-9_\\+:@\\.\\[\\]]+/cancel$")},

	&SkipRule{"GET", regexp.MustCompile("^/assets/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")},
	&SkipRule{"GET", regexp.MustCompile("^/assets/[a-zA-Z0-9_\\+:@\\.\\[\\]]+/offers$")},
}

// ServeHTTP handles incoming HTTP requests and attempt to authenticate them.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	withStatus := With(ctx, Status{AutStFailed, nil})

	username, password, _ := r.BasicAuth()
	skip := false
	for _, s := range SkipList {
		if s.Method == r.Method && s.Pattern.MatchString(r.URL.Path) {
			skip = true
		}
	}

	// Helper closure to fallback to the skiplist or log and return an
	// authentication error.
	failedAuth := func(err error) {
		if skip {
			withStatus = With(ctx, Status{AutStSkipped, nil})
			mint.Logf(ctx,
				"Authentication: status=%q username=%q",
				Get(withStatus).Status, username)
			m.Handler.ServeHTTP(w, r.WithContext(withStatus))
		} else {
			withStatus = With(ctx, Status{AutStFailed, nil})
			mint.Logf(ctx,
				"Authentication: status=%q username=%q",
				Get(withStatus).Status, username)
			respond.Error(withStatus, w, errors.Trace(err))
		}
	}

	user, err := model.LoadUserByUsername(ctx, username)
	if err != nil {
		failedAuth(errors.Trace(err))
		return
	} else if user == nil {
		failedAuth(errors.Trace(errors.NewUserErrorf(err,
			400, "username_invalid",
			"The username you are trying to authenticate with is not "+
				"associated with any existing user: %s.", username,
		)))
		return
	}

	if err := user.CheckPassword(ctx, password); err != nil {
		failedAuth(errors.Trace(errors.NewUserErrorf(err,
			400, "password_invalid", "The password you provided is invalid.",
		)))
		return
	}

	withStatus = With(ctx, Status{AutStSucceeded, user})
	mint.Logf(ctx,
		"Authentication: status=%q user=%q username=%q",
		Get(withStatus).Status, Get(withStatus).User.Token,
		username)

	m.Handler.ServeHTTP(w, r.WithContext(withStatus))
}

// Middleware that authenticates API requests.
func Middleware(h http.Handler) http.Handler {
	return middleware{h}
}
