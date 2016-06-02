package model

import (
	"time"

	"golang.org/x/net/context"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/token"
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
WHERE livemode = :livemode
  AND user_token = :user_token
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
		Livemode: livemode.Get(ctx),
		Username: username,
	}

	if rows, err := apidb.NamedQuery(`
SELECT *
FROM user_updates
WHERE livemode = :livemode
  AND username = :username
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
WHERE livemode = :livemode
  AND address = :address
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

// LoadActiveUserUpdateByAmbiguousID attempts to load an active user update
// with the given username, or user token, or address
func LoadActiveUserUpdateByAmbiguousID(
	ctx context.Context,
	ambiguousID string,
) (*UserUpdate, error) {
	update := UserUpdate{}

	if rows, err := apidb.NamedQuery(`
SELECT *
FROM user_updates
WHERE livemode = :livemode
  AND is_active = true
  AND (username = :id OR address = :id OR user_token = :id)
`, struct {
		Livemode bool
		ID       string
	}{
		livemode.Get(ctx),
		ambiguousID,
	}); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&update); err != nil {
		return nil, errors.Trace(err)
	}

	return &update, nil
}
