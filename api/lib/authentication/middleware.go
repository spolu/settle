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
	address, signature, _ := r.BasicAuth()
	challenge := r.Header.Get("Authorization-Challenge")
	skip := false
	for _, p := range SkipList {
		if r.URL.EscapedPath() == p {
			skip = true
		}
	}

	if skip {
		m.Handler.ServeHTTPC(ctx, w, r)
		return
	}

	// Check that the challenge is valid.
	err := CheckChallenge(ctx, challenge, RootLiveKeypair)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err))
		return
	}

	// Verify the challenge signature passed as basic auth.
	err = VerifyChallenge(ctx, challenge, address, signature)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err))
		return
	}

	// Check that the challenge was never used.
	auth, err := model.LoadAuthenticationByChallenge(ctx, challenge)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err))
		return
	} else if auth != nil {
		respond.Error(ctx, w, errors.NewUserError(err,
			400, "challenge_already_used",
			"The challenge you provided was already used. You must "+
				"resolve a new challenge for each API request.",
		))
		return
	}

	auth, err = model.CreateAuthentication(ctx,
		r.Method, r.URL.String(), challenge, address, signature)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err))
		return
	}

	logging.Logf(ctx,
		"Authentication Succeeded: skip=%t challenge=%q address=%q signature=%q",
		skip, challenge, address, signature)

	m.Handler.ServeHTTPC(ctx, w, r)
}

// Middleware that authenticates API requests.
func Middleware(h goji.Handler) goji.Handler {
	return middleware{h}
}
