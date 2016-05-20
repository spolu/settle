package auth

import (
	"net/http"

	"github.com/spolu/settl/model"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/logging"
	"github.com/spolu/settl/util/respond"

	"goji.io"

	"golang.org/x/net/context"
)

// SkipList is the list of endpoints that do not require authentication.
var SkipList = []string{
	"/challenges",
}

type authenticator struct {
	goji.Handler
}

// ServeHTTPC handles incoming HTTP requests and attempt to authenticate them.
func (a authenticator) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	address, signature, _ := r.BasicAuth()
	challenge := r.Header.Get("Challenge")
	skip := false
	for _, p := range SkipList {
		if r.URL.EscapedPath() == p {
			skip = true
		}
	}

	if skip {
		a.Handler.ServeHTTPC(ctx, w, r)
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

	a.Handler.ServeHTTPC(ctx, w, r)
}

// Authenticator is a middleware that authenticates API requests.
func Authenticator(h goji.Handler) goji.Handler {
	return authenticator{h}
}
