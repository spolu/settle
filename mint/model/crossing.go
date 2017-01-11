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
	Owner       string
	Token       string
	Created     time.Time
	Propagation mint.PgType

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
		Propagation:    crossing.Propagation,
		Offer:          crossing.Offer,
		Amount:         (*big.Int)(&crossing.Amount),
		Status:         crossing.Status,
		Transaction:    crossing.Transaction,
		TransactionHop: crossing.Hop,
	}
}

// CreateCanonicalCrossing creates and stores a new Crossing object.
func CreateCanonicalCrossing(
	ctx context.Context,
	owner string,
	offer string,
	amount Amount,
	status mint.TxStatus,
	transaction string,
	hop int8,
) (*Crossing, error) {
	crossing := Crossing{
		Owner:       owner,
		Token:       token.New("crossing"),
		Created:     time.Now().UTC(),
		Propagation: mint.PgTpCanonical,

		Offer:  offer,
		Amount: amount,

		Status:      status,
		Transaction: transaction,
		Hop:         hop,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO crossings
  (owner, token, created, propagation, offer, amount, status, txn, hop)
VALUES
  (:owner, :token, :created, :propagation, :offer, :amount, :status, :txn, :hop)
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

// ID returns the ID of the object.
func (c *Crossing) ID() string {
	return fmt.Sprintf("%s[%s]", c.Owner, c.Token)
}

// Save updates the object database representation with the in-memory values.
func (c *Crossing) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx, "mint")
	_, err := sqlx.NamedExec(ext, `
UPDATE crossings
SET status = :status
WHERE owner = :owner
  AND token = :token
`, c)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadCanonicalCrossingByTransactionHop attempts to load the crossing for the
// given transaction and hop.
func LoadCanonicalCrossingByTransactionHop(
	ctx context.Context,
	transaction string,
	hop int8,
) (*Crossing, error) {
	crossing := Crossing{
		Transaction: transaction,
		Hop:         hop,
		Propagation: mint.PgTpCanonical,
	}

	ext := db.Ext(ctx, "mint")
	if rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM crossings
WHERE txn = :txn
  AND hop = :hop
  AND propagation = :propagation
`, crossing); err != nil {
		return nil, errors.Trace(err)
	} else if !rows.Next() {
		return nil, nil
	} else if err := rows.StructScan(&crossing); err != nil {
		defer rows.Close()
		return nil, errors.Trace(err)
	} else if err := rows.Close(); err != nil {
		return nil, errors.Trace(err)
	}

	return &crossing, nil
}

// LoadCanonicalCrossingsByTransaction loads all crossings that are associated
// with the specified transaction.
func LoadCanonicalCrossingsByTransaction(
	ctx context.Context,
	transaction string,
) ([]*Crossing, error) {
	query := map[string]interface{}{
		"txn":         transaction,
		"propagation": mint.PgTpCanonical,
	}

	ext := db.Ext(ctx, "mint")
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM crossings
WHERE txn = :txn
  AND propagation = :propagation
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
