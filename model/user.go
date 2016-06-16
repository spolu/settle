package model

import (
	"encoding/base64"
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
	"golang.org/x/crypto/scrypt"
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

	Username     string
	PasswordHash string `db:"password_hash"`
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

// CreateUser creates and stores a new User object.
func CreateUser(
	ctx context.Context,
	username string,
	password string,
) (*User, error) {
	user := User{
		Token:    token.New("user"),
		Livemode: livemode.Get(ctx),

		Username: username,
	}

	h, err := scrypt.Key([]byte(password), []byte(user.Token), 16384, 8, 1, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}

	user.PasswordHash = base64.StdEncoding.EncodeToString(h)

	if rows, err := mintDB.NamedQuery(`
INSERT INTO users
  (token, livemode, username, password_hash)
VALUES
  (:token, :livemode, :username, :password_hash)
RETURNING created
`, user); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, errors.Newf("Nothing returned from INSERT.")
	} else if err := rows.StructScan(&user); err != nil {
		return nil, errors.Trace(err)
	}

	return &user, nil
}

// CheckPassword checks if the provided password matches the password hash
// associated with that user.
func (u *User) CheckPassword(
	ctx context.Context,
	password string,
) error {
	h, err := scrypt.Key([]byte(password), []byte(u.Token), 16384, 8, 1, 64)
	if err != nil {
		return errors.Trace(err)
	}

	if u.PasswordHash != base64.StdEncoding.EncodeToString(h) {
		return errors.Newf("Password mismatch")
	}
	return nil
}

// UpdatePassword updates the password for this user by recomputing the
// password hash with the new password provided.
func (u *User) UpdatePassword(
	ctx context.Context,
	password string,
) error {
	h, err := scrypt.Key([]byte(password), []byte(u.Token), 16384, 8, 1, 64)
	if err != nil {
		return errors.Trace(err)
	}

	u.PasswordHash = base64.StdEncoding.EncodeToString(h)

	if _, err := mintDB.NamedQuery(`
UPDATE users SET password_hash = :password_hash
WHERE token = :token
`, u); err != nil {
		return errors.Trace(err)
	}

	return nil
}