package db

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// ContextKey is the type of the key used with context to carry the contextual
// db and transaction.
type ContextKey string

const (
	// mapKey the context.Context key to store the db map.
	mapKey ContextKey = "db.map"
)

// WithDB stores the db in the provided context under tag.
func WithDB(
	ctx context.Context,
	tag string,
	db *sqlx.DB,
) context.Context {
	m := map[string]*sqlx.DB{}
	if ctx.Value(mapKey) != nil {
		m = ctx.Value(mapKey).(map[string]*sqlx.DB)
	}
	m[tag] = db
	return context.WithValue(ctx, mapKey, m)
}

// GetDB returns the db currently stored in the context under tag.
func GetDB(
	ctx context.Context,
	tag string,
) *sqlx.DB {
	m := ctx.Value(mapKey).(map[string]*sqlx.DB)
	if db, ok := m[tag]; ok {
		return db
	}
	return nil
}

// GetDBMap returns the db map currently stored in the context.
func GetDBMap(
	ctx context.Context,
) map[string]*sqlx.DB {
	return ctx.Value(mapKey).(map[string]*sqlx.DB)
}

// WithDBMap stores the db map in the provided context.
func WithDBMap(
	ctx context.Context,
	m map[string]*sqlx.DB,
) context.Context {
	return context.WithValue(ctx, mapKey, m)
}
