package model

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settl/lib/errors"
)

// UserUpdate represents an update to a user object. The most recent UserUpdate
// represents the current user.
type UserUpdate struct {
	ID       int64
	Token    string
	Created  time.Time
	Livemode bool

	Creation      time.Time
	Username      string
	Address       string
	EncryptedSeed string

	Email    string
	Verifier string
}

var insertUserUpdate *sqlx.NamedStmt
var findLatestUserUpdateByToken *sqlx.NamedStmt
var findLatestUserUpdateByUsername *sqlx.NamedStmt

func init() {
	ensureAPIDB()
	err := error(nil)

	insertUserUpdate, err = apidb.PrepareNamed(`
INSERT INTO user_updates
  (token, livemode, creation, username, address, encrypted_seed, email,
  verifier)
VALUES
  (:token, :livemode, :creation, :username, :address, :encrypted_seed, :email,
  :verifier)
RETURNING id, created
`)
	if err != nil {
		log.Fatal(errors.Details(err))
	}
}
