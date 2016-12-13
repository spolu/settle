// OWNER: stan

package model

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"golang.org/x/crypto/scrypt"

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

	Expiry time.Time
	Lock   string
	Secret *string
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
		Created:     transaction.Created.UnixNano() / mint.TimeResolutionNs,
		Owner:       transaction.Owner,
		Propagation: transaction.Propagation,
		Pair: fmt.Sprintf("%s/%s",
			transaction.BaseAsset, transaction.QuoteAsset),
		Amount:      (*big.Int)(&transaction.Amount),
		Destination: transaction.Destination,
		Path:        []string(transaction.Path),
		Expiry:      transaction.Expiry.UnixNano() / mint.TimeResolutionNs,
		Status:      transaction.Status,
		Lock:        transaction.Lock,
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
	owner string,
	baseAsset string,
	quoteAsset string,
	amount Amount,
	destination string,
	path []string,
	status mint.TxStatus,
	expiry time.Time,
) (*Transaction, error) {
	tok := token.New("transaction")

	secret := token.RandStr()
	h, err := scrypt.Key([]byte(secret), []byte(tok), 16384, 8, 1, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}
	lock := base64.StdEncoding.EncodeToString(h)

	transaction := Transaction{
		Owner:       owner,
		Token:       tok,
		Created:     time.Now().UTC(),
		Propagation: mint.PgTpCanonical,

		BaseAsset:   baseAsset,
		QuoteAsset:  quoteAsset,
		Amount:      amount,
		Destination: destination,
		Path:        OfPath(path),
		Status:      status,

		Expiry: expiry.UTC(),
		Lock:   lock,
		Secret: &secret,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO transactions
  (owner, token, created, propagation, base_asset, quote_asset,
   amount, destination, path, status, expiry, lock, secret)
VALUES
  (:owner, :token, :created, :propagation, :base_asset, :quote_asset,
   :amount, :destination, :path, :status, :expiry, :lock, :secret)
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
	token string,
	created time.Time,
	owner string,
	baseAsset string,
	quoteAsset string,
	amount Amount,
	destination string,
	path []string,
	status mint.TxStatus,
	expiry time.Time,
	lock string,
) (*Transaction, error) {
	transaction := Transaction{
		Owner:       owner,
		Token:       token,
		Created:     created.UTC(),
		Propagation: mint.PgTpPropagated,

		BaseAsset:   baseAsset,
		QuoteAsset:  quoteAsset,
		Amount:      amount,
		Destination: destination,
		Path:        OfPath(path),
		Status:      status,
		Expiry:      expiry.UTC(),
		Lock:        lock,
		Secret:      nil,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO transactions
  (owner, token, created, propagation, base_asset, quote_asset,
   amount, destination, path, status, expiry, lock, secret)
VALUES
  (:owner, :token, :created, :propagation, :base_asset, :quote_asset,
   :amount, :destination, :path, :status, :expiry, :lock, :secret)
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

// Save updates the object database representation with the in-memory values.
func (t *Transaction) Save(
	ctx context.Context,
) error {
	t.Expiry = t.Expiry.UTC()
	ext := db.Ext(ctx)
	_, err := sqlx.NamedExec(ext, `
UPDATE transactions
SET status = :status, expiry = :expiry
WHERE owner = :owner
  AND token = :token
`, t)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
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

// LoadPropagatedTransactionByOwnerToken attempts to load the propagated
// transaction for the given owner and token.
func LoadPropagatedTransactionByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Transaction, error) {
	transaction := Transaction{
		Owner:       owner,
		Token:       token,
		Propagation: mint.PgTpPropagated,
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

// LoadTransactionByID attempts to load the transaction (canonical or
// propagated) for the given owner and token.
func LoadTransactionByID(
	ctx context.Context,
	id string,
) (*Transaction, error) {
	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	transaction := Transaction{
		Owner: owner,
		Token: token,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM transactions
WHERE owner = :owner
  AND token = :token
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
