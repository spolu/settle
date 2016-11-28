// OWNER: stan

package model

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/mint"
)

// Transaction represents a transaction across a chain of offers.
type Transaction struct {
	User        string
	Owner       string
	Token       string
	Created     time.Time
	Propagation mint.PgType

	BaseAsset   string `db:"base_asset"`  // BaseAsset name.
	QuoteAsset  string `db:"quote_asset"` // QuoteAsset name.
	Amount      Amount
	Destination string
	Path        OfPath

	Status mint.TxStatus
}

// NewTransactionResource generates a new resource.
func NewTransactionResource(
	ctx context.Context,
	transaction *Transaction,
	operations []*Operation,
	crossings []*Crossing,
) mint.TransactionResource {
	tx := mint.TransactionResource{
		ID: fmt.Sprintf(
			"%s[%s]", transaction.Owner, transaction.Token),
		Created: transaction.Created.UnixNano() / (1000 * 1000),
		Owner:   transaction.Owner,
		Pair: fmt.Sprintf("%s/%s",
			transaction.BaseAsset, transaction.QuoteAsset),
		Amount:      (*big.Int)(&transaction.Amount),
		Destination: transaction.Destination,
		Path:        []string(transaction.Path),
		Status:      transaction.Status,
		Operations:  []mint.OperationResource{},
		Crossings:   []mint.CrossingResource{},
	}
	for _, op := range operations {
		tx.Operations = append(tx.Operations, NewOperationResource(ctx, op))
	}
	for _, cr := range crossings {
		tx.Crossings = append(tx.Crossings, NewCrossingResource(ctx, cr))
	}
	return tx
}

// CreateCanonicalTransaction creates and stores a new canonical Transaction
// object.
func CreateCanonicalTransaction(
	ctx context.Context,
	user string,
	owner string,
	baseAsset string,
	quoteAsset string,
	amount Amount,
	destination string,
	path []string,
	status mint.TxStatus,
) (*Transaction, error) {
	transaction := Transaction{
		User:        user,
		Owner:       owner,
		Token:       token.New("transaction"),
		Created:     time.Now(),
		Propagation: mint.PgTpCanonical,

		BaseAsset:   baseAsset,
		QuoteAsset:  quoteAsset,
		Amount:      amount,
		Destination: destination,
		Path:        OfPath(path),
		Status:      status,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO transactions
  (user, owner, token, created, propagation, base_asset, quote_asset,
   amount, destination, path, status)
VALUES
  (:user, :owner, :token, :created, :propagation, :base_asset, :quote_asset,
   :amount, :destination, :path, :status)
`, transaction); err != nil {
		switch err := err.(type) {
		case *pq.Error:
			if err.Code.Name() == "unique_violation" {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		case sqlite3.Error:
			if err.ExtendedCode == sqlite3.ErrConstraintUnique {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		}
		return nil, errors.Trace(err)
	}

	return &transaction, nil
}

// CreatePropagatedTransaction creates and stores a new propagated Transaction
// object.
func CreatePropagatedTransaction(
	ctx context.Context,
	user string,
	token string,
	created time.Time,
	owner string,
	baseAsset string,
	quoteAsset string,
	amount Amount,
	destination string,
	path []string,
	status mint.TxStatus,
) (*Transaction, error) {
	transaction := Transaction{
		User:        user,
		Owner:       owner,
		Token:       token,
		Created:     created,
		Propagation: mint.PgTpPropagated,

		BaseAsset:   baseAsset,
		QuoteAsset:  quoteAsset,
		Amount:      amount,
		Destination: destination,
		Path:        OfPath(path),
		Status:      status,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO transactions
  (user, owner, token, created, propagation, base_asset, quote_asset,
   amount, destination, path, status)
VALUES
  (:user, :owner, :token, :created, :propagation, :base_asset, :quote_asset,
   :amount, :destination, :path, :status)
`, transaction); err != nil {
		switch err := err.(type) {
		case *pq.Error:
			if err.Code.Name() == "unique_violation" {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		case sqlite3.Error:
			if err.ExtendedCode == sqlite3.ErrConstraintUnique {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		}
		return nil, errors.Trace(err)
	}

	return &transaction, nil
}

// LoadCanonicalTransactionByOwnerToken attempts to load the canonical
// transaction for the given owner and token.
func LoadCanonicalTransactionByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Transaction, error) {
	transaction := Transaction{
		Owner:       owner,
		Token:       token,
		Propagation: mint.PgTpCanonical,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM transactions
WHERE owner = :owner
  AND token = :token
  AND propagation = :propagation
`, transaction); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&transaction); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &transaction, nil
}
