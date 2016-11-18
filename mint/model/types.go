// OWNER: stan

package model

import (
	"database/sql/driver"
	"math/big"

	"github.com/spolu/settle/lib/errors"
)

// Amount extends big.Int to implement sql.Scanner and driver.Valuer.
type Amount big.Int

// Scan implements sql.Scanner.
func (b *Amount) Scan(src interface{}) error {
	switch src := src.(type) {
	case int64:
		(*big.Int)(b).SetInt64(src)
	case []byte:
		if _, success := (*big.Int)(b).SetString(string(src), 10); !success {
			return errors.Newf("Impossible to set Amount with string: %q", src)
		}
	case string:
		if _, success := (*big.Int)(b).SetString(src, 10); !success {
			return errors.Newf("Impossible to set Amount with string: %q", src)
		}
	default:
		return errors.Newf("Incompatible type for Amount with value: %q", src)
	}

	return nil
}

// Value implements driver.Valuer.
func (b Amount) Value() (value driver.Value, err error) {
	return (*big.Int)(&b).String(), nil
}

// PgType is the propagation type of an object.
type PgType string

const (
	//PgTpCanonical is an offer owned by this mint.
	PgTpCanonical PgType = "canonical"
	//PgTpPropagated is an offer propagated to this mint.
	PgTpPropagated PgType = "propagated"
)

// Value implements driver.Valuer
func (s PgType) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *PgType) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = PgType(src)
	case string:
		*s = PgType(src)
	default:
		return errors.Newf(
			"Incompatible type for PgType with value: %q", src)
	}

	return nil
}

// OfStatus is the status of an offer.
type OfStatus string

const (
	//OfStActive is used to mark an offer as active.
	OfStActive OfStatus = "active"
	//OfStClosed is used to mark an offer as closed.
	OfStClosed OfStatus = "closed"
)

// Value implements driver.Valuer.
func (s OfStatus) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *OfStatus) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = OfStatus(src)
	case string:
		*s = OfStatus(src)
	default:
		return errors.Newf(
			"Incompatible status for OfStatus with value: %q", src)
	}

	return nil
}

// UpStatus is the status of an offer.
type UpStatus string

const (
	//UpStPending is used to mark an update as pending.
	UpStPending UpStatus = "pending"
	//UpStSucceeded is used to mark an update as succeeded.
	UpStSucceeded UpStatus = "succeeded"
	//UpStFailed is used to mark an update as failed.
	UpStFailed UpStatus = "failed"
)

// Value implements driver.Valuer.
func (s UpStatus) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *UpStatus) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = UpStatus(src)
	case string:
		*s = UpStatus(src)
	default:
		return errors.Newf(
			"Incompatible status for UpStatus with value: %q", src)
	}

	return nil
}
