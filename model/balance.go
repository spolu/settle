// OWNER: stan

package model

import (
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/lib/tx"

	"golang.org/x/net/context"
)

// Balance represents a user balance for a given asset. Balances are updated as
// operations are created.
type Balance struct {
	Token    string
	Created  time.Time
	Livemode bool

	Asset string // Asset token.
	Owner string // Owner user address.
	Value big.Int
}

func init() {
	ensureMintDB()
}

// CreateBalance creates and store a new Balance object. Only one balance can
// exist for an asset, owner pair. Existing balance should be retrieved and
// updated instead.
func CreateBalance(
	ctx context.Context,
	asset string,
	owner string,
	value big.Int,
) (*Balance, error) {
	balance := Balance{
		Token:    token.New("balance"),
		Livemode: livemode.Get(ctx),

		Asset: asset,
		Owner: owner,
		Value: value,
	}

	ext := tx.Ext(ctx, MintDB())
	if rows, err := sqlx.NamedQuery(ext, `
INSERT INTO balances
  (token, livemode, asset, owner, value)
VALUES
  (:token, :livemode, :asset, :owner, :value)
RETURNING created
`, balance); err != nil {
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
	} else if err := rows.StructScan(&balance); err != nil {
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &balance, nil
}
