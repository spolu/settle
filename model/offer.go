// OWNER: stan

package model

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spolu/peer-currencies/lib/errors"
	"github.com/spolu/peer-currencies/lib/livemode"
	"github.com/spolu/peer-currencies/lib/token"
	"github.com/spolu/peer-currencies/lib/tx"
	"golang.org/x/net/context"
)

// Offer represents an offer for an asset pair. Offers are stored on the mint
// of the offer's owner (which acts as reference on its state). They are also
// stored indicatively on the mints of the offer's assets (as bid on one, as
// ask on the other) for discovery.
type Offer struct {
	Token    string
	Created  time.Time
	Livemode bool

	Owner      string // Owner user address.
	BaseAsset  string `db:"base_asset"`
	QuoteAsset string `db:"quote_asset"`

	Type OfType

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
	owner string,
	baseAsset string,
	quoteAsset string,
	oftype OfType,
	basePrice Amount,
	quotePrice Amount,
	amount Amount,
	status OfStatus,
) (*Offer, error) {
	offer := Offer{
		Token:    token.New("offer"),
		Livemode: livemode.Get(ctx),

		Owner:      owner,
		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		Type:       oftype,
		BasePrice:  basePrice,
		QuotePrice: quotePrice,
		Amount:     amount,
		Status:     status,
	}

	ext := tx.Ext(ctx, MintDB())
	if rows, err := sqlx.NamedQuery(ext, `
INSERT INTO offers
  (token, livemode, owner, base_asset, quote_asset, type, base_price,
   quote_price, amount, status)
VALUES
  (:token, :livemode, :owner, :base_asset, :quote_asset, :type, :base_price,
  :quote_price, :amount, :status)
RETURNING created
`, offer); err != nil {
		switch err := err.(type) {
		case *pq.Error:
			if err.Code.Name() == "unique_violation" {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		}
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, errors.Newf("Nothing returned from INSERT.")
	} else if err := rows.StructScan(&offer); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &offer, nil
}
