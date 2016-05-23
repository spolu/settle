package livemode

import (
	"net/http"

	"goji.io"

	"golang.org/x/net/context"
)

const (
	// livemodeKey the context.Context key to store the livemode.
	livemodeKey int = iota
	// livemodeHeader is the livemode header.
	livemodeHeader = "Livemode"
)

// With stores the livemode in the provided context.
func With(
	ctx context.Context,
	livemode bool,
) context.Context {
	return context.WithValue(ctx, livemodeKey, livemode)
}

// Get returns the livemode currently stored in the context
func Get(
	ctx context.Context,
) bool {
	return ctx.Value(livemodeKey).(bool)
}

type middleware struct {
	goji.Handler
}

// ServeHTTPC handles incoming HTTP requests and attempts to extract the
// livemode from it, defaulting to `false`.
func (m middleware) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	withLivemode := With(ctx, false)
	if lvm := r.Header.Get(livemodeHeader); lvm == "true" {
		withLivemode = With(ctx, true)
	}

	m.Handler.ServeHTTPC(withLivemode, w, r)
}

// Middleware that logs methods, URLs, remote addresses, status, lantency.
func Middleware(h goji.Handler) goji.Handler {
	return middleware{h}
}
