// OWNER: stan

package model

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/lib/env"
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
	User    string
	Owner   string
	Token   string
	Created time.Time

	Code  string // Asset code.
	Scale int8   // Asset scale.
}

// CreateAsset creates and stores a new Asset object.
func CreateAsset(
	ctx context.Context,
	owner string,
	code string,
	scale int8,
) (*Asset, error) {
	asset := Asset{
		User: authentication.Get(ctx).User.Token,
		Owner: fmt.Sprintf("%s@%s",
			authentication.Get(ctx).User.Username,
			env.GetMintHost(ctx)),
		Token:   token.New("asset"),
		Created: time.Now(),

		Code:  code,
		Scale: scale,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO assets
  (user, owner, token, created, code, scale)
VALUES
  (:user, :owner, :token, :created, :code, :scale)
`, asset); err != nil {
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

	return &asset, nil
}

// LoadAssetByOwnerCodeScale attempts to load an asset by its owner address,
// code and scale.
func LoadAssetByOwnerCodeScale(
	ctx context.Context,
	owner string,
	code string,
	scale int8,
) (*Asset, error) {
	asset := Asset{
		Owner: owner,
		Code:  code,
		Scale: scale,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM assets
WHERE owner = :owner
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
