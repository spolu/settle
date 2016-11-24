// OWNER: stan

package model

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
)

// Transaction represents a transaction across a chain of offers.
type Transaction struct {
	User        string
	Owner       string
	Token       string
	Created     time.Time
	Propagation PgType

	BaseAsset   string `db:"base_asset"`  // BaseAsset name.
	QuoteAsset  string `db:"quote_asset"` // QuoteAsset name.
	Amount      Amount
	Destination string
	Path        OfPath
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
) (*Transaction, error) {
	transaction := Transaction{
		User:        user,
		Owner:       owner,
		Token:       token.New("transaction"),
		Created:     time.Now(),
		Propagation: PgTpCanonical,

		BaseAsset:   baseAsset,
		QuoteAsset:  quoteAsset,
		Amount:      amount,
		Destination: destination,
		Path:        OfPath(path),
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO transactions
  (user, owner, token, created, propagation, base_asset, quote_asset,
   amount, destination, path)
VALUES
  (:user, :owner, :token, :created, :propagation, :base_asset, :quote_asset,
   :amount, :destination, :path)
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
		Propagation: PgTpCanonical,
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