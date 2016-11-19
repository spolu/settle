// OWNER: stan

package model

import (
	"context"
	"regexp"
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

// Save updates the object database representation with the in-memory values.
func (u *Update) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx)
	_, err := sqlx.NamedExec(ext, `
UPDATE updatess
SET status = :status
AND attempts = :attempts
WHERE token = :token
`, u)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadUpdateByToken attempts to load a user with the given user token.
func LoadUpdateByToken(
	ctx context.Context,
	token string,
) (*Update, error) {
	update := Update{
		Token: token,
	}

	ext := db.Ext(ctx)
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM updates
WHERE token = :token
`, update); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&update); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &update, nil
}

var offerTokenRegexp = regexp.MustCompile("^offer_[a-zA-Z0-9]+$")
var operationTokenRegexp = regexp.MustCompile("^operation_[a-zA-Z0-9]+$")

// Execute executes an update.
func (u *Update) Execute(
	ctx context.Context,
) error {
	switch {
	case offerTokenRegexp.MatchString(u.SubjectToken):
		switch u.Type {
		case PgTpCanonical:
		case PgTpPropagated:
		}
	case operationTokenRegexp.MatchString(u.SubjectToken):
		switch u.Type {
		case PgTpCanonical:
			return ExecutePropagatedOperationUpdate(ctx, u)
		case PgTpPropagated:
			return ExecutePropagatedOperationUpdate(ctx, u)
		}
	}
	return errors.Newf("Unknown update type or token")
}
