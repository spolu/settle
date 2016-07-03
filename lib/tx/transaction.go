package tx

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/peer-currencies/lib/logging"
	"github.com/spolu/peer-currencies/lib/token"
	"golang.org/x/net/context"
)

const (
	// transactionKey the context.Context key to store the current transaction.
	transactionKey string = "tx.transaction"
)

// Transaction stores the current mintDB transaction.
type Transaction struct {
	Tx    *sqlx.Tx
	Token string
}

// With stores the transaction in the provided context.
func With(
	ctx context.Context,
	transaction Transaction,
) context.Context {
	return context.WithValue(ctx, transactionKey, transaction)
}

// Get retrieves the current transaction form the context.
func Get(
	ctx context.Context,
) Transaction {
	return ctx.Value(transactionKey).(Transaction)
}

// Begin returns a new context with a new transaction set.
func Begin(
	ctx context.Context,
	db *sqlx.DB,
) context.Context {
	token := token.New("tx")
	logging.Logf(ctx,
		"Transaction: begin %s.", token)
	return With(ctx, Transaction{
		Tx:    db.MustBegin(),
		Token: token,
	})
}

// Commit commits the transaction in the current context.
func Commit(
	ctx context.Context,
) {
	logging.Logf(ctx,
		"Transaction: commit %s.", Get(ctx).Token)
	err := Get(ctx).Tx.Commit()
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
	err := Get(ctx).Tx.Rollback()
	if err != sql.ErrTxDone && err != nil {
		panic(err)
	} else if err == nil {
		logging.Logf(ctx,
			"Transaction: rollback %s.", Get(ctx).Token)
	}
}

// Ext returns the current Ext (a transaction if one has begin, or the DB
// otherwise).
func Ext(
	ctx context.Context,
	db *sqlx.DB,
) sqlx.Ext {
	if ctx.Value(transactionKey) != nil && Get(ctx).Tx != nil {
		return Get(ctx).Tx
	}
	return db
}
