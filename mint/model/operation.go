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

// MaxAssetAmount is the maximum amount for an asset (2^128).
var MaxAssetAmount = new(big.Int).Exp(
	new(big.Int).SetInt64(2), new(big.Int).SetInt64(128), nil)

// Operation represents a movement of an asset, either from an account to
// another. Asset owners can hold a balance in their own assets so operations
// referring to the asset owner are either issuing or annihilating the asset.
// - Canonical operations are stored on the mint of the operation's owner
//   (which acts as source of truth on its state).
// - Propagated operations are stored on the mints of the operation's source or
//   destination, for retrieval by impacted users.
// - When part of a transaction, an operation refers the transaction. Operation
//   created out of a transaction are created `settled`.
// - Only settled operation are propagated.
type Operation struct {
	User        string
	Owner       string // Owner address.
	Token       string
	Created     time.Time
	Propagation mint.PgType

	Asset       string // Asset name.
	Source      string // Source address (if owner, issuance).
	Destination string // Destination addres (if owner, annihilation).
	Amount      Amount

	Status      mint.TxStatus
	Transaction *string `db:"txn"`
}

// NewOperationResource generates a new resource.
func NewOperationResource(
	ctx context.Context,
	operation *Operation,
	asset *Asset,
) mint.OperationResource {
	return mint.OperationResource{
		ID: fmt.Sprintf(
			"%s[%s]", operation.Owner, operation.Token),
		Created:     operation.Created.UnixNano() / (1000 * 1000),
		Owner:       operation.Owner,
		Asset:       NewAssetResource(ctx, asset),
		Source:      operation.Source,
		Destination: operation.Destination,
		Amount:      (*big.Int)(&operation.Amount),
		Status:      operation.Status,
		Transaction: operation.Transaction,
	}
}

// CreateCanonicalOperation creates and stores a new Operation.
func CreateCanonicalOperation(
	ctx context.Context,
	user string,
	owner string,
	asset string,
	source string,
	destination string,
	amount Amount,
	status mint.TxStatus,
	transaction *string,
) (*Operation, error) {
	operation := Operation{
		User:        user,
		Owner:       owner,
		Token:       token.New("operation"),
		Created:     time.Now(),
		Propagation: mint.PgTpCanonical,

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,

		Status:      status,
		Transaction: transaction,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO operations
  (user, owner, token, created, propagation, asset, source, destination,
   amount, status, txn)
VALUES
  (:user, :owner, :token, :created, :propagation, :asset, :source, :destination,
   :amount, :status, :txn)
`, operation); err != nil {
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

	return &operation, nil
}

// CreatePropagatedOperation creates and stores a new Operation.
func CreatePropagatedOperation(
	ctx context.Context,
	user string,
	token string,
	created time.Time,
	owner string,
	asset string,
	source string,
	destination string,
	amount Amount,
) (*Operation, error) {
	operation := Operation{
		User:        user,
		Owner:       owner,
		Token:       token,
		Created:     created,
		Propagation: mint.PgTpPropagated,

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO operations
  (user, owner, token, created, propagation, asset, source, destination,
   amount, status, txn)
VALUES
  (:user, :owner, :token, :created, :propagation, :asset, :source, :destination,
   :amount, :status, :txn)
`, operation); err != nil {
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

	return &operation, nil
}

// LoadCanonicalOperationByOwnerToken attempts to load the canonical operation
// for the given owner and token.
func LoadCanonicalOperationByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Operation, error) {
	operation := Operation{
		Owner:       owner,
		Token:       token,
		Propagation: mint.PgTpCanonical,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM operations
WHERE owner = :owner
  AND token = :token
  AND propagation = :propagation
`, operation); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&operation); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &operation, nil
}
