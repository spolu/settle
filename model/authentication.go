package model

import (
	"time"

	"golang.org/x/net/context"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
)

// Authentication represents a sucessful authentication. It is used to ensure
// challenges are not used.
type Authentication struct {
	ID       int64
	Token    string
	Created  time.Time
	Livemode bool

	Method string
	URL    string

	Challenge string
	Address   string
	Signature string
}

func init() {
	ensureAPIDB()
}

// CreateAuthentication creates and stores a new Authentication object.
func CreateAuthentication(
	ctx context.Context,
	method string,
	url string,
	challenge string,
	address string,
	signature string,
) (*Authentication, error) {
	auth := Authentication{
		Token:    token.New("authentication"),
		Livemode: livemode.Get(ctx),

		Method: method,
		URL:    url,

		Challenge: challenge,
		Address:   address,
		Signature: signature,
	}

	tx := apidb.MustBegin()
	defer tx.Rollback()

	if rows, err := apidb.NamedQuery(`
INSERT INTO authentications
  (token, livemode, method, url, challenge, address, signature)
VALUES
  (:token, :livemode, :method, :url, :challenge, :address, :signature)
RETURNING id, created
`, auth); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, errors.Newf("Nothing returned from INSERT.")
	} else if err := rows.StructScan(&auth); err != nil {
		return nil, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Trace(err)
	}

	return &auth, nil
}

// LoadAuthenticationByChallenge attempts to load an Authentication by its
// challenge value.
func LoadAuthenticationByChallenge(
	ctx context.Context,
	challenge string,
) (*Authentication, error) {
	auth := Authentication{
		Challenge: challenge,
	}

	if rows, err := apidb.NamedQuery(`
SELECT *
FROM authentications
WHERE challenge = :challenge
`, auth); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&auth); err != nil {
		return nil, errors.Trace(err)
	}

	return &auth, nil
}
