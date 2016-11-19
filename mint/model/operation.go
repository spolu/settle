// OWNER: stan

package model

import (
	"context"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
)

// MaxAssetAmount is the maximum amount for an asset (2^128).
var MaxAssetAmount = new(big.Int).Exp(
	new(big.Int).SetInt64(2), new(big.Int).SetInt64(128), nil)

// Operation represents a movement of an asset, either from an account to
// another, or to an account only in the case of issuance. Amount is
// represented as a Amount and store in database as a NUMERIC(39).
// - Canonical operations are stored on the mint of the operation's owner
//   (which acts as source of truth on its state).
// - Propagated operations are indicatively stored on the mints of the
//   operation's source or destination, for retrieval by impacted users.
// - Operations are immutable.
type Operation struct {
	User    string
	Owner   string // Owner address.
	Token   string
	Created time.Time
	Type    PgType

	Asset       string  // Asset name.
	Source      *string // Source address (if nil issuance).
	Destination *string // Destination addres (if nil annihilation).
	Amount      Amount
}

// CreateCanonicalOperation creates and stores a new canonical Operation.
func CreateCanonicalOperation(
	ctx context.Context,
	user string,
	owner string,
	asset string,
	source *string,
	destination *string,
	amount Amount,
) (*Operation, error) {
	operation := Operation{
		User:    user,
		Owner:   owner,
		Token:   token.New("operation"),
		Created: time.Now(),
		Type:    PgTpCanonical,

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO operations
  (user, owner, token, created, type, asset, source, destination,
   amount)
VALUES
  (:user, :owner, :token, :created, :type, :asset, :source, :destination,
   :amount)
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

// CreatePropagatedOperation creates and stores a new propagated Operation.
func CreatePropagatedOperation(
	ctx context.Context,
	user string,
	owner string,
	token string,
	created time.Time,
	asset string,
	source *string,
	destination *string,
	amount Amount,
) (*Operation, error) {
	operation := Operation{
		User:    user,
		Owner:   owner,
		Token:   token,
		Created: created,
		Type:    PgTpPropagated,

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO operations
  (user, owner, token, created, type, asset, source, destination,
   amount)
VALUES
  (:user, :owner, :token, :created, :type, :asset, :source, :destination,
   :amount)
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
		Owner: owner,
		Token: token,
		Type:  PgTpCanonical,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM operations
WHERE owner = :owner
  AND token = :token
  AND type = :type
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
