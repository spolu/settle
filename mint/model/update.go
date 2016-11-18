// OWNER: stan

package model

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
)

// Update represents an update request from one mint to another. This is the
// mechanism used to propagate offers and operations across mints and maintain
// them updated.
// Updates don't have a User or Owner as they are not owned by a particular
// user but rather the mint itself. Also Token is not propagated (to avoid
// interference by malicious actors).
type Update struct {
	Token   string
	Created time.Time
	Type    PgType

	SubjectOwner string `db:"subject_owner"`
	SubjectToken string `db:"subject_token"`

	Source      string // Source mint host
	Destination string // Desination mint host

	Status   UpStatus // pending, succeeded, failed
	Attempts int
}

// CreateCanonicalUpdate creates and stores a new canonical update.
func CreateCanonicalUpdate(
	ctx context.Context,
	subjOwner string,
	subjToken string,
	source string,
	destination string,
) (*Update, error) {
	update := Update{
		Token:   token.New("update"),
		Created: time.Now(),
		Type:    PgTpCanonical,

		SubjectOwner: subjOwner,
		SubjectToken: subjToken,

		Source:      source,
		Destination: destination,

		Status:   UpStPending,
		Attempts: 0,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO updates
  (token, created, type, subject_owner, subject_token, source,
   destination, status, attempts)
VALUES
  (:token, :created, :type, :subject_owner, :subject_token, :source,
   :destination, :status, :attempts)
`, update); err != nil {
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

	return &update, nil
}

// CreatePropagatedUpdate creates and stores a new canonical update.
func CreatePropagatedUpdate(
	ctx context.Context,
	subjOwner string,
	subjToken string,
	source string,
	destination string,
) (*Update, error) {
	update := Update{
		Token:   token.New("update"),
		Created: time.Now(),
		Type:    PgTpCanonical,

		SubjectOwner: subjOwner,
		SubjectToken: subjToken,

		Source:      source,
		Destination: destination,

		Status:   UpStPending,
		Attempts: 0,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO updates
  (token, created, type, subject_owner, subject_token, source,
   destination, status, attempts)
VALUES
  (:token, :created, :type, :subject_owner, :subject_token, :source,
   :destination, :status, :attempts)
`, update); err != nil {
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

	return &update, nil
}

// Execute executes an update
func (u *Update) Execute(
	ctx context.Context,
) error {
	return nil
}
