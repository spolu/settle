package tx

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settle/model"
	"goji.io"
	"golang.org/x/net/context"
)

const (
	// transactionKey the context.Context key to store the current transaction.
	transactionKey string = "tx.transaction"
)

// Transaction stores the current mintDB transaction.
type Transaction struct {
	Tx *sqlx.Tx
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

type middleware struct {
	goji.Handler
}

// ServeHTTPC handles incoming HTTP requests and starts a new transaction for
// each of them, making sure to commit them when they are completed.
func (m middleware) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	withTransaction := With(ctx, Transaction{
		Tx: model.MintDB().MustBegin(),
	})

	//logging.Logf(ctx, "Livemode: livemode=%t", Get(withLivemode))

	m.Handler.ServeHTTPC(withTransaction, w, r)
}

// Middleware that logs methods, URLs, remote addresses, status, lantency.
func Middleware(h goji.Handler) goji.Handler {
	return middleware{h}
}
