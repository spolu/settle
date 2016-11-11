package db

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// ContextKey is the type of the key used with context to carry the contextual
// db and transaction.
type ContextKey string

const (
	// dbKey the context.Context key to store the db.
	dbKey ContextKey = "db.db"
)

// WithDB stores the db in the provided context.
func WithDB(
	ctx context.Context,
	db *sqlx.DB,
) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// GetDB returns the db currently stored in the context.
func GetDB(
	ctx context.Context,
) *sqlx.DB {
	return ctx.Value(dbKey).(*sqlx.DB)
}
