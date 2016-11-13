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

// Balance represents a user balance for a given asset. Balances are updated as
// operations are created.
type Balance struct {
	Token   string
	Created time.Time

	Asset string // Asset token.
	Owner string // Owner user address.
	Value Amount
}

// CreateBalance creates and store a new Balance object. Only one balance can
// exist for an asset, owner pair. Existing balance should be retrieved and
// updated instead.
func CreateBalance(
	ctx context.Context,
	asset string,
	owner string,
	value Amount,
) (*Balance, error) {
	balance := Balance{
		Token:   token.New("balance"),
		Created: time.Now(),

		Asset: asset,
		Owner: owner,
		Value: value,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO balances
  (token, created, asset, owner, value)
VALUES
  (:token, :created, :asset, :owner, :value)
`, balance); err != nil {
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

	return &balance, nil
}

// Save updates the object database representation with the in-memory values.
func (b *Balance) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx)
	_, err := sqlx.NamedExec(ext, `
UPDATE balances
SET value = :value
WHERE token = :token
`, b)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadBalanceByAssetOwner attempts to load a balance for the given asset token
// and owner address.
func LoadBalanceByAssetOwner(
	ctx context.Context,
	asset string,
	owner string,
) (*Balance, error) {
	balance := Balance{
		Asset: asset,
		Owner: owner,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM balances
WHERE asset = :asset
  AND owner = :owner
`, balance); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&balance); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &balance, nil
}

// LoadOrCreateBalanceByAssetOwner loads an existing balance for the specified
// asset and owner or creates one (with a 0 value) if it does not exist.
func LoadOrCreateBalanceByAssetOwner(
	ctx context.Context,
	asset string,
	owner string,
) (*Balance, error) {
	balance, err := LoadBalanceByAssetOwner(ctx, asset, owner)
	if err != nil {
		return nil, errors.Trace(err)
	} else if balance == nil {
		balance, err = CreateBalance(ctx, asset, owner, Amount{})
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return balance, nil
}
