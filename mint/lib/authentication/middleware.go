package authentication

import (
	"net/http"
	"regexp"

	"github.com/spolu/peer-currencies/lib/errors"
	"github.com/spolu/peer-currencies/lib/livemode"
	"github.com/spolu/peer-currencies/lib/logging"
	"github.com/spolu/peer-currencies/lib/respond"
	"github.com/spolu/peer-currencies/model"
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
	goji.Handler
}

// SkipRule defines a skip rule for authentication
type SkipRule struct {
	Method  string
	Pattern *regexp.Regexp
}

// SkipList is the list of endpoints that do not require authentication.
var SkipList = []*SkipRule{
	&SkipRule{"GET", regexp.MustCompile("^/users/[a-zA-Z0-9_]+$")},
	&SkipRule{"POST", regexp.MustCompile("^/offers$")},
}

// ServeHTTPC handles incoming HTTP requests and attempt to authenticate them.
func (m middleware) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	withStatus := With(ctx, Status{AutStFailed, nil})

	username, password, _ := r.BasicAuth()
	skip := false
	for _, s := range SkipList {
		if s.Method == r.Method && s.Pattern.MatchString(r.URL.EscapedPath()) {
			skip = true
		}
	}

	// Helper closure to fallback to the skiplist or log and return an
	// authentication error.
	failedAuth := func(err error) {
		if skip {
			withStatus = With(ctx, Status{AutStSkipped, nil})
			logging.Logf(ctx,
				"Authentication: status=%q livemode=%t username=%q",
				Get(withStatus).Status, livemode.Get(ctx), username)
			m.Handler.ServeHTTPC(withStatus, w, r)
		} else {
			withStatus = With(ctx, Status{AutStFailed, nil})
			logging.Logf(ctx,
				"Authentication: status=%q livemode=%t username=%q",
				Get(withStatus).Status, livemode.Get(ctx), username)
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
	logging.Logf(ctx,
		"Authentication: status=%q livemode=%t user=%q username=%q",
		Get(withStatus).Status, livemode.Get(ctx), Get(withStatus).User.Token,
		username)

	m.Handler.ServeHTTPC(withStatus, w, r)
}

// Middleware that authenticates API requests.
func Middleware(h goji.Handler) goji.Handler {
	return middleware{h}
}
