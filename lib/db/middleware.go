package db

import (
	"net/http"

	"github.com/jmoiron/sqlx"
)

type middleware struct {
	Handler http.Handler
	M       map[string]*sqlx.DB
}

// ServeHTTPC handles incoming HTTP requests and injects the current db.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	withMapDB := WithDBMap(ctx, m.M)
	m.Handler.ServeHTTP(w, r.WithContext(withMapDB))
}

// Middleware returns a middleware that injects the specified DB in requests.
func Middleware(
	m map[string]*sqlx.DB,
) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return middleware{h, m}
	}
}
