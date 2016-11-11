package env

import "net/http"

type middleware struct {
	http.Handler
	*Env
}

// ServeHTTPC handles incoming HTTP requests and injects the current Env in
// their context.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	withEnv := With(ctx, m.Env)
	m.Handler.ServeHTTP(w, r.WithContext(withEnv))
}

// Middleware returns a middleware that injects the specified Env in requests.
func Middleware(
	env *Env,
) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return middleware{h, env}
	}
}
