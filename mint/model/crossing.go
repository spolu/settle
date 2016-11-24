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
	"github.com/spolu/settle/mint"
)

// Crossing represents a transaction crossing an offer and consuming some of
// its amount. Crossings are not propagated.
type Crossing struct {
	User    string
	Owner   string
	Token   string
	Created time.Time

	Offer  string
	Amount Amount

	Status      mint.TxStatus
	Transaction string `db:"txn"`
}

// CreateCrossing creates and stores a new Crossing object.
func CreateCrossing(
	ctx context.Context,
	user string,
	owner string,
	offer string,
	amount Amount,
	status mint.TxStatus,
	transaction string,
) (*Crossing, error) {
	crossing := Crossing{
		User:    user,
		Owner:   owner,
		Token:   token.New("crossing"),
		Created: time.Now(),

		Offer:  offer,
		Amount: amount,

		Status:      status,
		Transaction: transaction,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO crossings
  (user, owner, token, created, offer, amount, status, txn)
VALUES
  (:user, :owner, :token, :created, :offer, :amount, :status, :txn)
`, crossing); err != nil {
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

	return &crossing, nil
}
