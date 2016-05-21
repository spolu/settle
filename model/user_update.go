package model

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// UserUpdate represents an update to a user object. The most recent UserUpdate
// represents the current user.
type UserUpdate struct {
	ID      int64
	Token   string
	Created time.Time

	Creation      time.Time
	Username      string
	EncryptedSeed string
}

var insertUserUpdate *sqlx.NamedStmt
var findLatestUserUpdateByToken *sqlx.NamedStmt
var findLatestUserUpdateByUsername *sqlx.NamedStmt

func init() {
	ensureAPIDB()
	/*
			err := error(nil)

			insertUserUpdate, err = apidb.PrepareNamed(`
		INSERT INTO user_updates
		  (token, creation, username, encrypted_seed)
		VALUES
		  (:token, :creation, :username, :encrypted_seed)
		RETURNING id, created
		`)
			if err != nil {
				log.Fatal(errors.Details(err))
			}
	*/
}
