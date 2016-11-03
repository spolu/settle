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

// Value implements driver.Valuer
func (b Amount) Value() (value driver.Value, err error) {
	return (*big.Int)(&b).String(), nil
}

// OfType is the type of an offer.
type OfType string

const (
	//OfTpCanonical is an offer owned by this mint.
	OfTpCanonical OfType = "canonical"
	//OfTpPropagated is an offer propagated to this mint.
	OfTpPropagated OfType = "propagated"
)

// Value implements driver.Valuer
func (s OfType) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *OfType) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = OfType(src)
	case string:
		*s = OfType(src)
	default:
		return errors.Newf(
			"Incompatible type for OfType with value: %q", src)
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

// Value implements driver.Valuer
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
