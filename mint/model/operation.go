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

// Operation represents a movement of an asset. Asset owners cannot hold a
// balance in their own assets so operations referring to the asset owner are
// either issuing or annihilating the asset.
// - Canonical operations are stored on the mint of the operation's owner
//   (which acts as source of truth on its state).
// - Propagated operations are stored on the mints of the operation's source or
//   destination, for retrieval by impacted users (only settled operations are
//   reserved).
// - When part of a transaction, an operation refers the transaction and hop.
type Operation struct {
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
	Hop         *int8   `db:"hop"`
}

// NewOperationResource generates a new resource.
func NewOperationResource(
	ctx context.Context,
	operation *Operation,
) mint.OperationResource {
	return mint.OperationResource{
		ID: fmt.Sprintf(
			"%s[%s]", operation.Owner, operation.Token),
		Created:        operation.Created.UnixNano() / mint.TimeResolutionNs,
		Owner:          operation.Owner,
		Propagation:    operation.Propagation,
		Asset:          operation.Asset,
		Source:         operation.Source,
		Destination:    operation.Destination,
		Amount:         (*big.Int)(&operation.Amount),
		Status:         operation.Status,
		Transaction:    operation.Transaction,
		TransactionHop: operation.Hop,
	}
}

// CreateCanonicalOperation creates and stores a new Operation.
func CreateCanonicalOperation(
	ctx context.Context,
	owner string,
	asset string,
	source string,
	destination string,
	amount Amount,
	status mint.TxStatus,
	transaction *string,
	hop *int8,
) (*Operation, error) {
	operation := Operation{
		Owner:       owner,
		Token:       token.New("operation"),
		Created:     time.Now().UTC(),
		Propagation: mint.PgTpCanonical,

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,

		Status:      status,
		Transaction: transaction,
		Hop:         hop,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO operations
  (owner, token, created, propagation, asset, source, destination,
   amount, status, txn, hop)
VALUES
  (:owner, :token, :created, :propagation, :asset, :source, :destination,
   :amount, :status, :txn, :hop)
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
	owner string,
	token string,
	created time.Time,
	asset string,
	source string,
	destination string,
	amount Amount,
	status mint.TxStatus,
	transaction *string,
	hop *int8,
) (*Operation, error) {
	operation := Operation{
		Owner:       owner,
		Token:       token,
		Created:     created.UTC(),
		Propagation: mint.PgTpPropagated,

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,

		Status:      status,
		Transaction: transaction,
		Hop:         hop,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO operations
  (owner, token, created, propagation, asset, source, destination,
   amount, status, txn, hop)
VALUES
  (:owner, :token, :created, :propagation, :asset, :source, :destination,
   :amount, :status, :txn, :hop)
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

// ID returns the ID of the object.
func (o *Operation) ID() string {
	return fmt.Sprintf("%s[%s]", o.Owner, o.Token)
}

// Save updates the object database representation with the in-memory values.
func (o *Operation) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx, "mint")
	_, err := sqlx.NamedExec(ext, `
UPDATE operations
SET status = :status
WHERE owner = :owner
  AND token = :token
`, o)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
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

	ext := db.Ext(ctx, "mint")
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

// LoadCanonicalOperationByID attempts to load the canonical operation for the
// given id.
func LoadCanonicalOperationByID(
	ctx context.Context,
	id string,
) (*Operation, error) {
	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return LoadCanonicalOperationByOwnerToken(ctx, owner, token)
}

// LoadCanonicalOperationByTransactionHop attempts to load the canonical
// operation for the given transaction and hop.
func LoadCanonicalOperationByTransactionHop(
	ctx context.Context,
	transaction string,
	hop int8,
) (*Operation, error) {
	operation := Operation{
		Transaction: &transaction,
		Hop:         &hop,
		Propagation: mint.PgTpCanonical,
	}

	ext := db.Ext(ctx, "mint")
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM operations
WHERE txn = :txn
  AND hop = :hop
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

// LoadCanonicalOperationsByTransaction loads all operations that are
// associated with the specified transaction.
func LoadCanonicalOperationsByTransaction(
	ctx context.Context,
	transaction string,
) ([]*Operation, error) {
	query := map[string]interface{}{
		"txn":         &transaction,
		"propagation": mint.PgTpCanonical,
	}

	ext := db.Ext(ctx, "mint")
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM operations
WHERE txn = :txn
  AND propagation = :propagation
`, query)
	if err != nil {
		return nil, errors.Trace(err)
	}

	operations := []*Operation{}

	defer rows.Close()
	for rows.Next() {
		op := Operation{}
		err := rows.StructScan(&op)
		if err != nil {
			return nil, errors.Trace(err)
		}
		operations = append(operations, &op)
	}

	return operations, nil
}

// LoadPropagatedOperationByOwnerToken attempts to load the propagated
// operation for the given owner and token.
func LoadPropagatedOperationByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Operation, error) {
	operation := Operation{
		Owner:       owner,
		Token:       token,
		Propagation: mint.PgTpPropagated,
	}

	ext := db.Ext(ctx, "mint")
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
