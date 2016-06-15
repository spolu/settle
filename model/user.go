package model

import (
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"golang.org/x/net/context"
)

// User represents a user object. User objects are not managed by the mint and
// solely accesed in read-only mode, leaving user management to an external
// system with access to the same underlying mintDB.
type User struct {
	ID       int64
	Token    string
	Created  time.Time
	Livemode bool

	Username string
	Email    string
	SCrypt   string
}

func init() {
	ensureMintDB()
}

// LoadUserByToken attempts to load a user with the given user token.
func LoadUserByToken(
	ctx context.Context,
	token string,
) (*User, error) {
	user := User{
		Token:    token,
		Livemode: livemode.Get(ctx),
	}

	if rows, err := mintDB.NamedQuery(`
SELECT *
FROM users
WHERE livemode = :livemode
  AND token = :token
`, user); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&user); err != nil {
		return nil, errors.Trace(err)
	}

	return &user, nil
}

// LoadUserByUsername attempts to load a user with the given username.
func LoadUserByUsername(
	ctx context.Context,
	username string,
) (*User, error) {
	user := User{
		Username: username,
		Livemode: livemode.Get(ctx),
	}

	if rows, err := mintDB.NamedQuery(`
SELECT *
FROM users
WHERE livemode = :livemode
  AND username = :username
`, user); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&user); err != nil {
		return nil, errors.Trace(err)
	}

	return &user, nil
}
