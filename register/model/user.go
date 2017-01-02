// OWNER: stan

package model

import (
	"context"
	"encoding/base64"
	"time"

	"golang.org/x/crypto/scrypt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/register"
)

// User represents a user object. Users are not tied to a mint user until they
// are verified.
type User struct {
	Token   string
	Created time.Time

	Status   register.UsrStatus
	Username string
	Email    string

	Secret   string
	Password string

	MintToken *string `db:"mint_token"`
}

// NewUserResource generates a new user resource.
func NewUserResource(
	ctx context.Context,
	user *User,
) register.UserResource {
	return register.UserResource{
		ID:       user.Token,
		Created:  user.Created.UnixNano() / register.TimeResolutionNs,
		Status:   user.Status,
		Username: user.Username,
		Email:    user.Email,
	}
}

// CreateUser creates and stores a new User object.
func CreateUser(
	ctx context.Context,
	username string,
	email string,
) (*User, error) {
	user := User{
		Token:   token.New("user"),
		Created: time.Now().UTC(),

		Status:   register.UsrStUnverified,
		Username: username,
		Email:    email,
	}

	h, err := scrypt.Key([]byte(token.RandStr()), []byte(user.Token), 16384, 8, 1, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}
	user.Secret = base64.URLEncoding.EncodeToString(h)

	h, err = scrypt.Key([]byte(token.RandStr()), []byte(user.Token), 16384, 8, 1, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}
	user.Password = base64.URLEncoding.EncodeToString(h)

	ext := db.Ext(ctx, "register")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO users
  (token, created, status, username, email, secret, password)
VALUES
  (:token, :created, :status, :username, :email, :secret, :password)
`, user); err != nil {
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

	return &user, nil
}

// Save updates the object database representation with the in-memory values.
func (u *User) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx, "register")
	_, err := sqlx.NamedExec(ext, `
UPDATE users
SET status = :status, password = :password
WHERE token = :token
`, u)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadUserByUsername attempts to load a user with the given username.
func LoadUserByUsername(
	ctx context.Context,
	username string,
) (*User, error) {
	user := User{
		Username: username,
	}

	ext := db.Ext(ctx, "register")
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM users
WHERE username = :username
`, user); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&user); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &user, nil
}
