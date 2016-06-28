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

// Operation represents a movement of an asset, either from an account to
// another, or to an account only in the case of issuance. Amount is
// represented as a BigInt and store in database as a NUMERIC(39).
type Operation struct {
	Token    string
	Create   time.Time
	Livemode bool

	Asset       string  // Asset token.
	Source      *string // Source user address.
	Destination string  // Destination user addres.
	Amount      BigInt
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
	amount BigInt,
) (*Operation, error) {
	operation := Operation{
		Token:    token.New("operation"),
		Livemode: livemode.Get(ctx),

		Asset:       asset,
		Source:      source,
		Destination: destination,
		Amount:      amount,
	}

	ext := tx.Ext(ctx, MintDB())
	if rows, err := sqlx.NamedQuery(ext, `
INSERT INTO operations
  (token, livemode, asset, source, destination, amount)
VALUES
  (:token, :livemode, :asset, :source, :destination, :amount)
RETURNING created
`, operation); err != nil {
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
	} else if err := rows.StructScan(&operation); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &operation, nil
}
