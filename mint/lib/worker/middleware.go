package worker

import "net/http"

type middleware struct {
	http.Handler
	*Worker
}

// ServeHTTPC handles incoming HTTP requests and injects the current Worker in
// their context.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	withWorker := With(ctx, m.Worker)
	m.Handler.ServeHTTP(w, r.WithContext(withWorker))
}

// Middleware returns a middleware that injects the specified Worker in
// requests.
func Middleware(
	worker *Worker,
) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return middleware{h, worker}
	}
}
