package db

import (
	"net/http"

	"github.com/jmoiron/sqlx"
)

type middleware struct {
	http.Handler
	*sqlx.DB
}

// ServeHTTPC handles incoming HTTP requests and injects the current db.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	withDB := WithDB(ctx, m.DB)
	m.Handler.ServeHTTP(w, r.WithContext(withDB))
}

// Middleware returns a middleware that injects the specified DB in requests.
func Middleware(
	db *sqlx.DB,
) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return middleware{h, db}
	}
}
