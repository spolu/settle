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

// Balance represents a user balance for a given asset. Balances are updated as
// operations are created and then propagated.
// - Canonical balances are stored on the mint of the balance's asset owner
//   (which acts as source of truth on its state).
// - Propagated balances are indicatively stored on the mints of the balances's
//   holders.
type Balance struct {
	Owner       string
	Token       string
	Created     time.Time
	Propagation mint.PgType

	Asset  string // Asset name.
	Holder string // Holder address.
	Value  Amount
}

// NewBalanceResource generates a new resource.
func NewBalanceResource(
	ctx context.Context,
	balance *Balance,
) mint.BalanceResource {
	return mint.BalanceResource{
		ID: fmt.Sprintf(
			"%s[%s]", balance.Owner, balance.Token),
		Created:     balance.Created.UnixNano() / mint.TimeResolutionNs,
		Owner:       balance.Owner,
		Propagation: balance.Propagation,
		Asset:       balance.Asset,
		Holder:      balance.Holder,
		Value:       (*big.Int)(&balance.Value),
	}
}

// CreateCanonicalBalance creates and store a new Balance object. Only one
// balance can exist for an asset, holder pair.
func CreateCanonicalBalance(
	ctx context.Context,
	owner string,
	asset string,
	holder string,
	value Amount,
) (*Balance, error) {
	balance := Balance{
		Owner:       owner,
		Token:       token.New("balance"),
		Created:     time.Now().UTC(),
		Propagation: mint.PgTpCanonical,

		Asset:  asset,
		Holder: holder,
		Value:  value,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO balances
  (owner, token, created, propagation, asset, holder, value)
VALUES
  (:owner, :token, :created, :propagation, :asset, :holder, :value)
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

// CreatePropagatedBalance creates and store a new propagated Balance object.
func CreatePropagatedBalance(
	ctx context.Context,
	owner string,
	token string,
	created time.Time,
	asset string,
	holder string,
	value Amount,
) (*Balance, error) {
	balance := Balance{
		Owner:       owner,
		Token:       token,
		Created:     created.UTC(),
		Propagation: mint.PgTpPropagated,

		Asset:  asset,
		Holder: holder,
		Value:  value,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO balances
  (owner, token, created, propagation, asset, holder, value)
VALUES
  (:owner, :token, :created, :propagation, :asset, :holder, :value)
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

// ID returns the ID of the object.
func (b *Balance) ID() string {
	return fmt.Sprintf("%s[%s]", b.Owner, b.Token)
}

// Save updates the object database representation with the in-memory values.
func (b *Balance) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx, "mint")
	_, err := sqlx.NamedExec(ext, `
UPDATE balances
SET value = :value
WHERE owner = :owner
  AND token = :token
`, b)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadCanonicalBalanceByAssetHolder attempts to load a balance for the given
// holder address and asset name.
func LoadCanonicalBalanceByAssetHolder(
	ctx context.Context,
	asset string,
	holder string,
) (*Balance, error) {
	balance := Balance{
		Asset:       asset,
		Holder:      holder,
		Propagation: mint.PgTpCanonical,
	}

	ext := db.Ext(ctx, "mint")
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM balances
WHERE asset = :asset
  AND holder = :holder
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

// LoadOrCreateCanonicalBalanceByAssetHolder loads an existing balance for the
// specified asset and holder or creates one (with a 0 value) if it does not
// exist.
func LoadOrCreateCanonicalBalanceByAssetHolder(
	ctx context.Context,
	owner string,
	asset string,
	holder string,
) (*Balance, error) {
	balance, err := LoadCanonicalBalanceByAssetHolder(ctx, asset, holder)
	if err != nil {
		return nil, errors.Trace(err)
	} else if balance == nil {
		balance, err = CreateCanonicalBalance(ctx,
			owner, asset, holder, Amount{})
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return balance, nil
}

// LoadCanonicalBalanceByOwnerToken attempts to load a canonical balance for
// the given owner and token.
func LoadCanonicalBalanceByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Balance, error) {
	balance := Balance{
		Owner:       owner,
		Token:       token,
		Propagation: mint.PgTpCanonical,
	}

	ext := db.Ext(ctx, "mint")
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM balances
WHERE owner = :owner
  AND token = :token
  AND propagation = :propagation
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

// LoadCanonicalBalanceByID attempts to load the canonical balanceffer for the
// given id.
func LoadCanonicalBalanceByID(
	ctx context.Context,
	id string,
) (*Balance, error) {
	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return LoadCanonicalBalanceByOwnerToken(ctx, owner, token)
}

// LoadPropagatedBalanceByOwnerToken attempts to load a propagated balance for
// the given owner and token.
func LoadPropagatedBalanceByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Balance, error) {
	balance := Balance{
		Owner:       owner,
		Token:       token,
		Propagation: mint.PgTpPropagated,
	}

	ext := db.Ext(ctx, "mint")
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM balances
WHERE owner = :owner
  AND token = :token
  AND propagation = :propagation
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

// LoadBalanceListByHolder loads a balance list by holder.
func LoadBalanceListByHolder(
	ctx context.Context,
	createdBefore time.Time,
	limit uint,
	holder string,
) ([]Balance, error) {
	query := map[string]interface{}{
		"holder":         holder,
		"created_before": createdBefore.UTC(),
		"limit":          limit,
	}

	ext := db.Ext(ctx, "mint")
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM balances
WHERE holder = :holder
AND created < :created_before
ORDER BY created DESC
LIMIT :limit
`, query)
	if err != nil {
		return nil, errors.Trace(err)
	}

	balances := []Balance{}

	defer rows.Close()
	for rows.Next() {
		b := Balance{}
		err := rows.StructScan(&b)
		if err != nil {
			return nil, errors.Trace(err)
		}

		balances = append(balances, b)
	}

	return balances, nil
}

// LoadBalanceListByAsset loads a balance list by asset.
func LoadBalanceListByAsset(
	ctx context.Context,
	createdBefore time.Time,
	limit uint,
	asset string,
) ([]Balance, error) {
	query := map[string]interface{}{
		"asset":          asset,
		"created_before": createdBefore.UTC(),
		"limit":          limit,
	}

	ext := db.Ext(ctx, "mint")
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM balances
WHERE asset = :asset
AND created < :created_before
ORDER BY created DESC
LIMIT :limit
`, query)
	if err != nil {
		return nil, errors.Trace(err)
	}

	balances := []Balance{}

	defer rows.Close()
	for rows.Next() {
		b := Balance{}
		err := rows.StructScan(&b)
		if err != nil {
			return nil, errors.Trace(err)
		}

		balances = append(balances, b)
	}

	return balances, nil
}
