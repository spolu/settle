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

// Offer represents an offer for an asset pair.
// - Offers are always represented as asks
//   (ask on pair A/B offer to sell A (base asset) for B (quote asset)).
//   Amounts are expressed in quote asset.
// - Canonical offers are stored on the mint of the offer's owner (which acts
//   as source of truth on its state).
// - Propagated offers are indicatively stored on the mints of the offers's
//   assets, to compute order books.
type Offer struct {
	User        string
	Owner       string
	Token       string
	Created     time.Time
	Propagation PgType

	BaseAsset  string `db:"base_asset"`  // BaseAsset name.
	QuoteAsset string `db:"quote_asset"` // QuoteAsset name.
	BasePrice  Amount `db:"base_price"`
	QuotePrice Amount `db:"quote_price"`
	Amount     Amount

	Status    OfStatus
	Remainder Amount
}

// CreateCanonicalOffer creates and stores a new canonical Offer object.
func CreateCanonicalOffer(
	ctx context.Context,
	user string,
	owner string,
	baseAsset string,
	quoteAsset string,
	basePrice Amount,
	quotePrice Amount,
	amount Amount,
	status OfStatus,
	remainder Amount,
) (*Offer, error) {
	offer := Offer{
		User:        user,
		Owner:       owner,
		Token:       token.New("offer"),
		Created:     time.Now(),
		Propagation: PgTpCanonical,

		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,

		Status:    status,
		Remainder: remainder,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO offers
  (user, owner, token, created, propagation, base_asset, quote_asset,
   base_price, quote_price, amount, status, remainder)
VALUES
  (:user, :owner, :token, :created, :propagation, :base_asset, :quote_asset,
   :base_price, :quote_price, :amount, :status, :remainder)
`, offer); err != nil {
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

	return &offer, nil
}

// CreatePropagatedOffer creates and stores a new propagated Offer object.
func CreatePropagatedOffer(
	ctx context.Context,
	user string,
	owner string,
	token string,
	created time.Time,
	baseAsset string,
	quoteAsset string,
	basePrice Amount,
	quotePrice Amount,
	amount Amount,
	status OfStatus,
	remainder Amount,
) (*Offer, error) {
	offer := Offer{
		User:        user,
		Owner:       owner,
		Token:       token,
		Created:     created,
		Propagation: PgTpPropagated,

		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,

		Status:    status,
		Remainder: remainder,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO offers
  (user, owner, token, created, propagation, base_asset, quote_asset,
   base_price, quote_price, amount, status)
VALUES
  (:user, :owner, :token, :created, :propagation, :base_asset, :quote_asset,
   :base_price, :quote_price, :amount, :status, :remainder)
`, offer); err != nil {
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

	return &offer, nil
}

// Save updates the object database representation with the in-memory values.
func (o *Offer) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx)
	_, err := sqlx.NamedExec(ext, `
UPDATE offers
SET status = :status
WHERE user = :user
  AND owner = :owner
  AND token = :token
`, o)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadCanonicalOfferByOwnerToken attempts to load the canonical offer for the
// given owner and token.
func LoadCanonicalOfferByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Offer, error) {
	offer := Offer{
		Owner:       owner,
		Token:       token,
		Propagation: PgTpCanonical,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM offers
WHERE owner = :owner
  AND token = :token
  AND propagation = :propagation
`, offer); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&offer); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &offer, nil
}
