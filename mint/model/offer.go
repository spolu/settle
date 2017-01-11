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

// Offer represents an offer for an asset pair.
// - Offers are always represented as asks
//   (ask on pair A/B offer to sell A (base asset) for B (quote asset)).
//   Amounts are expressed in quote asset.
// - Canonical offers are stored on the mint of the offer's owner (which acts
//   as source of truth on its state).
// - Propagated offers are indicatively stored on the mints of the offers's
//   assets, to compute order books.
type Offer struct {
	Owner       string
	Token       string
	Created     time.Time
	Propagation mint.PgType

	BaseAsset  string `db:"base_asset"`  // BaseAsset name.
	QuoteAsset string `db:"quote_asset"` // QuoteAsset name.
	BasePrice  Amount `db:"base_price"`
	QuotePrice Amount `db:"quote_price"`
	Amount     Amount

	Status    mint.OfStatus
	Remainder Amount
}

// NewOfferResource generates a new resource.
func NewOfferResource(
	ctx context.Context,
	offer *Offer,
) mint.OfferResource {
	return mint.OfferResource{
		ID: fmt.Sprintf(
			"%s[%s]", offer.Owner, offer.Token),
		Created:     offer.Created.UnixNano() / mint.TimeResolutionNs,
		Owner:       offer.Owner,
		Propagation: offer.Propagation,
		Pair:        fmt.Sprintf("%s/%s", offer.BaseAsset, offer.QuoteAsset),
		Price: fmt.Sprintf(
			"%s/%s",
			(*big.Int)(&offer.BasePrice).String(),
			(*big.Int)(&offer.QuotePrice).String()),
		Amount:    (*big.Int)(&offer.Amount),
		Status:    offer.Status,
		Remainder: (*big.Int)(&offer.Remainder),
	}
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
	status mint.OfStatus,
	remainder Amount,
) (*Offer, error) {
	offer := Offer{
		Owner:       owner,
		Token:       token.New("offer"),
		Created:     time.Now().UTC(),
		Propagation: mint.PgTpCanonical,

		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,

		Status:    status,
		Remainder: remainder,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO offers
  (owner, token, created, propagation, base_asset, quote_asset,
   base_price, quote_price, amount, status, remainder)
VALUES
  (:owner, :token, :created, :propagation, :base_asset, :quote_asset,
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
	owner string,
	token string,
	created time.Time,
	baseAsset string,
	quoteAsset string,
	basePrice Amount,
	quotePrice Amount,
	amount Amount,
	status mint.OfStatus,
	remainder Amount,
) (*Offer, error) {
	offer := Offer{
		Owner:       owner,
		Token:       token,
		Created:     created.UTC(),
		Propagation: mint.PgTpPropagated,

		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,

		Status:    status,
		Remainder: remainder,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO offers
  (owner, token, created, propagation, base_asset, quote_asset,
   base_price, quote_price, amount, status, remainder)
VALUES
  (:owner, :token, :created, :propagation, :base_asset, :quote_asset,
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

// ID returns the ID of the object.
func (o *Offer) ID() string {
	return fmt.Sprintf("%s[%s]", o.Owner, o.Token)
}

// Save updates the object database representation with the in-memory values.
func (o *Offer) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx, "mint")
	_, err := sqlx.NamedExec(ext, `
UPDATE offers
SET status = :status, remainder = :remainder
WHERE owner = :owner
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
		Propagation: mint.PgTpCanonical,
	}

	ext := db.Ext(ctx, "mint")
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

// LoadCanonicalOfferByID attempts to load the canonical offer for the given
// id.
func LoadCanonicalOfferByID(
	ctx context.Context,
	id string,
) (*Offer, error) {
	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return LoadCanonicalOfferByOwnerToken(ctx, owner, token)
}

// LoadPropagatedOfferByOwnerToken attempts to load the propagated offer for
// the given owner and token.
func LoadPropagatedOfferByOwnerToken(
	ctx context.Context,
	owner string,
	token string,
) (*Offer, error) {
	offer := Offer{
		Owner:       owner,
		Token:       token,
		Propagation: mint.PgTpPropagated,
	}

	ext := db.Ext(ctx, "mint")
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

// LoadPropagatedOfferByID attempts to load the propagated offer for the given
// id.
func LoadPropagatedOfferByID(
	ctx context.Context,
	id string,
) (*Offer, error) {
	owner, token, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return LoadPropagatedOfferByOwnerToken(ctx, owner, token)
}

// LoadOfferListByBaseAsset loads a balance list by base asset.
func LoadOfferListByBaseAsset(
	ctx context.Context,
	createdBefore time.Time,
	limit uint,
	asset string,
) ([]Offer, error) {
	query := map[string]interface{}{
		"base_asset":     asset,
		"created_before": createdBefore.UTC(),
		"limit":          limit,
	}

	ext := db.Ext(ctx, "mint")
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM offers
WHERE base_asset = :base_asset
  AND created < :created_before
ORDER BY created DESC
LIMIT :limit
`, query)
	if err != nil {
		return nil, errors.Trace(err)
	}

	offers := []Offer{}

	defer rows.Close()
	for rows.Next() {
		o := Offer{}
		err := rows.StructScan(&o)
		if err != nil {
			return nil, errors.Trace(err)
		}

		offers = append(offers, o)
	}

	return offers, nil
}

// LoadOfferListByQuoteAsset loads a balance list by quote asset.
func LoadOfferListByQuoteAsset(
	ctx context.Context,
	createdBefore time.Time,
	limit uint,
	asset string,
) ([]Offer, error) {
	query := map[string]interface{}{
		"quote_asset":    asset,
		"created_before": createdBefore.UTC(),
		"limit":          limit,
	}

	ext := db.Ext(ctx, "mint")
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM offers
WHERE quote_asset = :quote_asset
  AND created < :created_before
ORDER BY created DESC
LIMIT :limit
`, query)
	if err != nil {
		return nil, errors.Trace(err)
	}

	offers := []Offer{}

	defer rows.Close()
	for rows.Next() {
		o := Offer{}
		err := rows.StructScan(&o)
		if err != nil {
			return nil, errors.Trace(err)
		}

		offers = append(offers, o)
	}

	return offers, nil
}
