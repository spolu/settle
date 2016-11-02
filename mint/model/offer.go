// OWNER: stan

package model

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/lib/tx"
	"golang.org/x/net/context"
)

// Offer represents an offer for an asset pair.
// - Offers are always represented as asks (ask on pair A/B offer to sell A for
//   B).
// - Canonical offers are stored on the mint of the offer's owner (which acts
//   as source of truth on its state).
// - Canonical offers's base price must be in an asset issued by the owner of
//   the offer.
// Non canonical offers are indicatively stored on the mints of the offers's
// assets, to compute order books.
type Offer struct {
	Token    string
	Created  time.Time
	Livemode bool

	Canonical bool // The offer was created on this mint.

	Owner      string // Owner user address.
	BaseAsset  string `db:"base_asset"`
	QuoteAsset string `db:"quote_asset"`

	BasePrice  Amount `db:"base_price"`
	QuotePrice Amount `db:"quote_price"`
	Amount     Amount

	Status OfStatus
}

func init() {
	ensureMintDB()
}

// CreateOffer creates and stores a new Offer object.
func CreateOffer(
	ctx context.Context,
	canonical bool,
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

		Canonical:  canonical,
		Owner:      owner,
		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,
		Status:     status,
	}

	ext := tx.Ext(ctx, MintDB())
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO offers
  (token, livemode, created, canonical, owner, base_asset, quote_asset,
   base_price, quote_price, amount, status)
VALUES
  (:token, :livemode, :created, :canonical, :owner, :base_asset, :quote_asset,
   :base_price, :quote_price, :amount, :status)
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
