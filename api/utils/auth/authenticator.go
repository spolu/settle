package auth

import (
	"net/http"

	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/logging"
	"github.com/spolu/settl/util/respond"

	"goji.io"

	"golang.org/x/net/context"
)

// SkipList is the list of endpoints that do not require authentication.
var SkipList = []string{
	"/tokens",
}

func init() {
}

type authenticator struct {
	goji.Handler
}

// ServeHTTPC handles incoming HTTP requests and attempt to authenticate them.
func (rl authenticator) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	address, signature, _ := r.BasicAuth()
	token := r.PostFormValue("token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	skip := false
	for _, p := range SkipList {
		if r.URL.EscapedPath() == p {
			skip = true
		}
	}

	logging.Logf(ctx,
		"Authenticator: skip=%t path=%q token=%q address=%q signature=%q",
		skip, r.URL.EscapedPath(), token, address, signature)

	if skip {
		rl.Handler.ServeHTTPC(ctx, w, r)
		return
	}

	// check that the token is valid
	err := CheckToken(ctx, token, RootLiveKeypair)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err))
		return
	}

	rl.Handler.ServeHTTPC(ctx, w, r)
}

// Authenticator is a middleware that authenticates API requests.
func Authenticator(h goji.Handler) goji.Handler {
	return authenticator{h}
}
