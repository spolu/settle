package model

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
	"golang.org/x/crypto/scrypt"
)

// User represents a user object. User objects are not managed by the mint and
// solely accesed in read-only mode, leaving user management to an external
// system with access to the same underlying mintDB.
type User struct {
	Token   string
	Created time.Time

	Username     string
	PasswordHash string `db:"password_hash"`
}

// CreateUser creates and stores a new User object.
func CreateUser(
	ctx context.Context,
	username string,
	password string,
) (*User, error) {
	user := User{
		Token:   token.New("user"),
		Created: time.Now().UTC(),

		Username: username,
	}

	h, err := scrypt.Key([]byte(password), []byte(user.Token), 16384, 8, 1, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}

	user.PasswordHash = base64.StdEncoding.EncodeToString(h)

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO users
  (token, created, username, password_hash)
VALUES
  (:token, :created, :username, :password_hash)
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
	ext := db.Ext(ctx, "mint")
	_, err := sqlx.NamedExec(ext, `
UPDATE users
SET username = :username, password_hash = :password_hash
WHERE token = :token
`, u)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadUserByToken attempts to load a user with the given user token.
func LoadUserByToken(
	ctx context.Context,
	token string,
) (*User, error) {
	user := User{
		Token: token,
	}

	ext := db.Ext(ctx, "mint")
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM users
WHERE token = :token
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

// LoadUserByUsername attempts to load a user with the given username.
func LoadUserByUsername(
	ctx context.Context,
	username string,
) (*User, error) {
	user := User{
		Username: username,
	}

	ext := db.Ext(ctx, "mint")
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

// UpdatePassword updates the password hash in memory using the provided
// password value.
func (u *User) UpdatePassword(
	ctx context.Context,
	password string,
) error {
	h, err := scrypt.Key([]byte(password), []byte(u.Token), 16384, 8, 1, 64)
	if err != nil {
		return errors.Trace(err)
	}

	u.PasswordHash = base64.StdEncoding.EncodeToString(h)

	return nil
}
