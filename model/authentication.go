package model

import (
	"database/sql"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settl/lib/errors"
	"github.com/spolu/settl/lib/livemode"
	"github.com/spolu/settl/lib/token"
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

var insertAuthentication *sqlx.NamedStmt
var findAuthenticationByChallenge *sqlx.NamedStmt

func init() {
	ensureAPIDB()
	err := error(nil)

	insertAuthentication, err = apidb.PrepareNamed(`
INSERT INTO authentications
  (token, livemode, method, url, challenge, address, signature)
VALUES
  (:token, :livemode, :method, :url, :challenge, :address, :signature)
RETURNING id, created
`)
	findAuthenticationByChallenge, err = apidb.PrepareNamed(`
SELECT *
FROM authentications
WHERE challenge = :challenge
`)
	if err != nil {
		log.Fatal(errors.Details(err))
	}
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

	row := insertAuthentication.QueryRowx(auth)
	if err := row.Err(); err != nil {
		return nil, errors.Trace(err)
	}
	err := row.StructScan(&auth)
	if err != nil {
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

	row := findAuthenticationByChallenge.QueryRowx(auth)
	if err := row.Err(); err != nil {
		return nil, errors.Trace(err)
	}
	err := row.StructScan(&auth)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &auth, nil
}
