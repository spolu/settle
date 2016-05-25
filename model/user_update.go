package model

import (
	"time"

	"golang.org/x/net/context"

	"github.com/spolu/settl/lib/errors"
	"github.com/spolu/settl/lib/livemode"
	"github.com/spolu/settl/lib/token"
)

// UserUpdate represents an update to a user object. The most recent UserUpdate
// represents the current user.
type UserUpdate struct {
	ID       int64
	Token    string
	Created  time.Time
	Livemode bool

	IsActive bool `db:"is_active"`

	UserToken     string `db:"user_token"`
	Creation      time.Time
	Username      string
	Address       string
	EncryptedSeed string `db:"encrypted_seed"`

	Email    string
	Verifier string
}

func init() {
	ensureAPIDB()
}

// CreateUserUpdate creates and stores a new active UserUpdate object, marking
// all previous user update for the same user token as inactive.
func CreateUserUpdate(
	ctx context.Context,
	userToken string,
	creation time.Time,
	username string,
	address string,
	encryptedSeed string,
	email string,
	verifier string,
) (*UserUpdate, error) {

	update := UserUpdate{
		Token:    token.New("user_update"),
		Livemode: livemode.Get(ctx),

		IsActive:      true,
		UserToken:     userToken,
		Creation:      creation,
		Username:      username,
		Address:       address,
		EncryptedSeed: encryptedSeed,

		Email:    email,
		Verifier: verifier,
	}

	tx := apidb.MustBegin()
	defer tx.Rollback()

	if _, err := tx.NamedExec(`
UPDATE user_updates
SET is_active = false
WHERE user_token = :user_token
  AND is_active = true
`, update); err != nil {
		return nil, errors.Trace(err)
	}

	if rows, err := apidb.NamedQuery(`
INSERT INTO user_updates
  (token, livemode, creation, is_active, user_token, username, address,
  encrypted_seed, email, verifier)
VALUES
  (:token, :livemode, :creation, :is_active, :user_token, :username, :address,
  :encrypted_seed, :email, :verifier)
RETURNING id, created
`, update); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, errors.Newf("Nothing returned from INSERT.")
	} else if err := rows.StructScan(&update); err != nil {
		return nil, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Trace(err)
	}

	return &update, nil
}

// LoadActiveUserUpdateByUserToken attempts to load an active user update with
// the given user token.
func LoadActiveUserUpdateByUserToken(
	ctx context.Context,
	userToken string,
) (*UserUpdate, error) {
	update := UserUpdate{
		UserToken: userToken,
	}

	if rows, err := apidb.NamedQuery(`
SELECT *
FROM user_updates
WHERE user_token = :user_token
  AND is_active = true
`, update); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&update); err != nil {
		return nil, errors.Trace(err)
	}

	return &update, nil
}

// LoadActiveUserUpdateByUsername attempts to load an active user update with
// the given username.
func LoadActiveUserUpdateByUsername(
	ctx context.Context,
	username string,
) (*UserUpdate, error) {
	update := UserUpdate{
		Username: username,
	}

	if rows, err := apidb.NamedQuery(`
SELECT *
FROM user_updates
WHERE username = :username
  AND is_active = true
`, update); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&update); err != nil {
		return nil, errors.Trace(err)
	}

	return &update, nil
}

// LoadActiveUserUpdateByAddress attempts to load an active user update with
// the given address.
func LoadActiveUserUpdateByAddress(
	ctx context.Context,
	address string,
) (*UserUpdate, error) {
	update := UserUpdate{
		Address: address,
	}

	if rows, err := apidb.NamedQuery(`
SELECT *
FROM user_updates
WHERE address = :address
  AND is_active = true
`, update); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&update); err != nil {
		return nil, errors.Trace(err)
	}

	return &update, nil
}
