package model

import (
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
	"golang.org/x/net/context"
)

// Operation represents a movement of an asset, either from an account to
// another, or to an account only in the case of issuance. Amount is
// represented as a big.Int and store in database as a NUMERIC(39).
type Operation struct {
	Token    string
	Create   time.Time
	Livemode bool

	Asset       string
	Source      *string
	Destination string
	Amount      big.Int
}

func init() {
	ensureMintDB()
}

// CreateOperation creates and stores a new Operation object. It also
// atomically checks and adjusts the balances mutated by this operation.
func CreateOperation(
	ctx context.Context,
	asset string,
	source *string,
	destination string,
	amount big.Int,
) (*Operation, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, errors.Trace(err)
	}
	var ext sqlx.Ext = tx

	operation := Operation{
		Token:    token.New("asset"),
		Livemode: livemode.Get(ctx),

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,
	}

	if rows, err := ext.NamedQuery(`
INSERT INTO operations
  (token, livemode, issuer, code, scale)
VALUES
  (:token, :livemode, :issuer, :code, :scale)
RETURNING created
`, asset); err != nil {
		switch err := err.(type) {
		case *pq.Error:
			if err.Code.Name() == "unique_violation" {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		default:
			return nil, errors.Trace(err)
		}
	} else if !rows.Next() {
		return nil, errors.Newf("Nothing returned from INSERT.")
	} else if err := rows.StructScan(&asset); err != nil {
		return nil, errors.Trace(err)
	}

	return &asset, nil
}
