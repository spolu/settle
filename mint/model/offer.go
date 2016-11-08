// OWNER: stan

package model

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/lib/tx"
)

// Offer represents an offer for an asset pair.
// - Offers are always represented as asks
//   (ask on pair A/B offer to sell A for B).
// - Canonical offers are stored on the mint of the offer's owner (which acts
//   as source of truth on its state).
// - Propagated offers are indicatively stored on the mints of the offers's
//   assets, to compute order books.
type Offer struct {
	Token    string
	Created  time.Time
	Livemode bool

	Owner      string // Owner user address.
	BaseAsset  string `db:"base_asset"`
	QuoteAsset string `db:"quote_asset"`

	BasePrice  Amount `db:"base_price"`
	QuotePrice Amount `db:"quote_price"`
	Amount     Amount

	Type   OfType
	Status OfStatus
}

func init() {
	ensureMintDB()
}

// CreateCanonicalOffer creates and stores a new canonical Offer object.
func CreateCanonicalOffer(
	ctx context.Context,
	owner string,
	baseAsset string,
	quoteAsset string,
	basePrice Amount,
	quotePrice Amount,
	amount Amount,
	status OfStatus,
) (*Offer, error) {
	offer := Offer{
		Token:    token.New("offer"),
		Livemode: livemode.Get(ctx),
		Created:  time.Now(),

		Owner:      owner,
		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,
		Type:       OfTpCanonical,
		Status:     status,
	}

	ext := tx.Ext(ctx, MintDB())
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO offers
  (token, livemode, created, owner, base_asset, quote_asset, base_price,
   quote_price, amount, type, status)
VALUES
  (:token, :livemode, :created, :owner, :base_asset, :quote_asset, :base_price, 
   :quote_price, :amount, :type, :status)
`, offer); err != nil {
		switch err := err.(type) {
		case *pq.Error:
			if err.Code.Name() == "unique_violation" {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		}
		return nil, errors.Trace(err)
	}

	return &offer, nil
}

// CreatePropagatedOffer creates and stores a new canonical Offer object.
func CreatePropagatedOffer(
	ctx context.Context,
	token string,
	created time.Time,
	owner string,
	baseAsset string,
	quoteAsset string,
	basePrice Amount,
	quotePrice Amount,
	amount Amount,
	status OfStatus,
) (*Offer, error) {
	offer := Offer{
		Token:    token,
		Livemode: livemode.Get(ctx),
		Created:  created,

		Owner:      owner,
		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,
		Type:       OfTpPropagated,
		Status:     status,
	}

	ext := tx.Ext(ctx, MintDB())
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO offers
  (token, livemode, created, owner, base_asset, quote_asset, base_price,
   quote_price, amount, type, status)
VALUES
  (:token, :livemode, :created, :owner, :base_asset, :quote_asset, :base_price, 
   :quote_price, :amount, :type, :status)
`, offer); err != nil {
		switch err := err.(type) {
		case *pq.Error:
			if err.Code.Name() == "unique_violation" {
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
	ext := tx.Ext(ctx, MintDB())
	_, err := sqlx.NamedExec(ext, `
UPDATE offers
SET owner = :owner, base_asset = :base_asset, quote_asset = :quote_asset,
    quote_asset = :quote_asset, base_price = :base_price,
	quote_price = :quote_price, amount = :amount, type = :type,
	status = :status
WHERE token = :token
`, o)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadOfferByToken attempts to load an offer for the given token.
func LoadOfferByToken(
	ctx context.Context,
	token string,
) (*Offer, error) {
	offer := Offer{
		Livemode: livemode.Get(ctx),
		Token:    token,
	}

	ext := tx.Ext(ctx, MintDB())
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM offers
WHERE livemode = :livemode
  AND token = :token
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
