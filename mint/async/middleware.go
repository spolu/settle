package async

import "net/http"

type middleware struct {
	http.Handler
	*Async
}

// ServeHTTPC handles incoming HTTP requests and injects the current Async in
// their context.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	withAsync := With(ctx, m.Async)
	m.Handler.ServeHTTP(w, r.WithContext(withAsync))
}

// Middleware returns a middleware that injects the specified Async in
// requests.
func Middleware(
	async *Async,
) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return middleware{h, async}
	}
}
