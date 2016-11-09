package livemode

import (
	"context"
	"net/http"

	"github.com/spolu/settle/lib/logging"
)

// ContextKey is the type of the key used with context to carry contextual
// livemode.
type ContextKey string

const (
	// livemodeKey the context.Context key to store the livemode.
	livemodeKey ContextKey = "livemode.livemode"
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
	http.Handler
}

// ServeHTTPC handles incoming HTTP requests and attempts to extract the
// livemode from it, defaulting to `false`.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	withLivemode := With(ctx, false)
	if lvm := r.Header.Get(livemodeHeader); lvm == "true" {
		withLivemode = With(ctx, true)
	}

	logging.Logf(ctx, "Livemode: livemode=%t", Get(withLivemode))

	m.Handler.ServeHTTP(w, r.WithContext(withLivemode))
}

// Middleware that extracts and inject the current livemode in the context.
func Middleware(h http.Handler) http.Handler {
	return middleware{h}
}
