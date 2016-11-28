// OWNER: stan

package model

import (
	"context"
	"fmt"
	"math/big"
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
	Hop         int8   `db:"hop"`
}

// NewCrossingResource generates a new resource.
func NewCrossingResource(
	ctx context.Context,
	crossing *Crossing,
) mint.CrossingResource {
	return mint.CrossingResource{
		ID: fmt.Sprintf(
			"%s[%s]", crossing.Owner, crossing.Token),
		Created:        crossing.Created.UnixNano() / mint.TimeResolutionNs,
		Owner:          crossing.Owner,
		Offer:          crossing.Offer,
		Amount:         (*big.Int)(&crossing.Amount),
		Status:         crossing.Status,
		Transaction:    crossing.Transaction,
		TransactionHop: crossing.Hop,
	}
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
	hop int8,
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
		Hop:         hop,
	}

	ext := db.Ext(ctx)
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO crossings
  (user, owner, token, created, offer, amount, status, txn, hop)
VALUES
  (:user, :owner, :token, :created, :offer, :amount, :status, :txn, :hop)
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

// LoadCrossingsByTransaction loads all crossings that are associated with the
// specified transaction.
func LoadCrossingsByTransaction(
	ctx context.Context,
	transaction string,
) ([]*Crossing, error) {
	query := Crossing{
		Transaction: transaction,
	}

	ext := db.Ext(ctx)
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM crossings
WHERE txn = :txn
`, query)
	if err != nil {
		return nil, errors.Trace(err)
	}

	crossings := []*Crossing{}

	defer rows.Close()
	for rows.Next() {
		cr := Crossing{}
		err := rows.StructScan(&cr)
		if err != nil {
			return nil, errors.Trace(err)
		}
		crossings = append(crossings, &cr)
	}

	return crossings, nil
}
