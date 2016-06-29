// OWNER: stan

package model

import (
	"regexp"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/lib/tx"
	"golang.org/x/net/context"
)

const (
	// AssetMinScale is the minimal value for an asset scale.
	AssetMinScale int8 = 0
	// AssetMaxScale is the minimal value for an asset scale.
	AssetMaxScale int8 = 24
)

// AssetCodeRegexp is used to validate asset codes at creation.
var AssetCodeRegexp = regexp.MustCompile("^[A-Z0-9\\-]{1,64}$")

// Asset represents an asset object. Asset are created by users (issuer).
type Asset struct {
	Token    string
	Created  time.Time
	Livemode bool

	Issuer string // Issuer user token.
	Code   string // Asset code.
	Scale  int8   // Asset scale.
}

func init() {
	ensureMintDB()
}

// CreateAsset creates and stores a new Asset object.
func CreateAsset(
	ctx context.Context,
	issuer string,
	code string,
	scale int8,
) (*Asset, error) {
	asset := Asset{
		Token:    token.New("asset"),
		Livemode: livemode.Get(ctx),

		Issuer: issuer,
		Code:   code,
		Scale:  scale,
	}

	ext := tx.Ext(ctx, MintDB())
	if rows, err := sqlx.NamedQuery(ext, `
INSERT INTO assets
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
		}
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, errors.Newf("Nothing returned from INSERT.")
	} else if err := rows.StructScan(&asset); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &asset, nil
}

// Save updates the object database representation with the in-memory values.
func (u *Asset) Save(
	ctx context.Context,
) error {
	ext := tx.Ext(ctx, MintDB())
	rows, err := sqlx.NamedQuery(ext, `
UPDATE users SET issuer = :issuer, code = :code, scale = :scale
WHERE token = :token
`, u)
	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()

	return nil
}

// LoadAssetByIssuerCodeScale attempts to load an asset by its issuer token,
// code and scale.
func LoadAssetByIssuerCodeScale(
	ctx context.Context,
	issuer string,
	code string,
	scale int8,
) (*Asset, error) {
	asset := Asset{
		Livemode: livemode.Get(ctx),
		Issuer:   issuer,
		Code:     code,
		Scale:    scale,
	}

	ext := tx.Ext(ctx, MintDB())
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM assets
WHERE livemode = :livemode
  AND issuer = :issuer
  AND code = :code
  AND scale = :scale
`, asset); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&asset); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &asset, nil
}
