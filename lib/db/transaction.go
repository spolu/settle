package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/token"
)

const (
	// transactionKey the context.Context key to store the current transaction.
	transactionKey ContextKey = "db.transaction"
)

// Transaction stores the current mintDB transaction.
type Transaction struct {
	Tx    *sqlx.Tx
	Token string
}

// WithTransaction stores the transaction in the provided context.
func WithTransaction(
	ctx context.Context,
	transaction Transaction,
) context.Context {
	return context.WithValue(ctx, transactionKey, transaction)
}

// GetTransaction retrieves the current transaction form the context.
func GetTransaction(
	ctx context.Context,
) Transaction {
	return ctx.Value(transactionKey).(Transaction)
}

// Begin returns a new context with a new transaction set.
func Begin(
	ctx context.Context,
	tag string,
) context.Context {
	if ctx.Value(mapKey) == nil || GetDB(ctx, tag) == nil {
		panic(fmt.Sprintf("db: no DB in context for tag %s", tag))
	}
	if ctx.Value(transactionKey) != nil && GetTransaction(ctx).Tx != nil {
		panic(fmt.Sprintf(
			"db: re-entrant transaction %s (re-entrant tag: %s)",
			GetTransaction(ctx).Token, tag))
	}
	token := token.New("tx")
	logging.Logf(ctx,
		"Transaction BEGIN: tag=%s token=%s", tag, token)
	return WithTransaction(ctx, Transaction{
		Tx:    GetDB(ctx, tag).MustBegin(),
		Token: token,
	})
}

// Commit commits the transaction in the current context.
func Commit(
	ctx context.Context,
) {
	logging.Logf(ctx,
		"Transaction COMMIT: token=%s", GetTransaction(ctx).Token)
	err := GetTransaction(ctx).Tx.Commit()
	if err != nil {
		panic(err)
	}
}

// LoggedRollback logs a rollback a commit or another rollback didn't take
// place before this call. Used in general with defer right after calling
// `Begin`.
// ```
//   ctx = tx.Begin(ctx)
//   defer tx.LoggedRollback(ctx)
// ```
func LoggedRollback(ctx context.Context) {
	err := GetTransaction(ctx).Tx.Rollback()
	if err != sql.ErrTxDone && err != nil {
		panic(err)
	} else if err == nil {
		logging.Logf(ctx,
			"Transaction ROLLBACK: token=%s", GetTransaction(ctx).Token)
	}
}

// Ext returns the current Ext (a transaction if one has begin, or the DB
// otherwise).
func Ext(
	ctx context.Context,
	tag string,
) sqlx.Ext {
	if ctx.Value(transactionKey) != nil && GetTransaction(ctx).Tx != nil {
		return GetTransaction(ctx).Tx
	}
	return GetDB(ctx, tag)
}
