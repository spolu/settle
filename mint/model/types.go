// OWNER: stan

package model

import (
	"database/sql/driver"
	"math/big"
	"strings"

	"github.com/spolu/settle/lib/errors"
)

// Amount extends big.Int to implement sql.Scanner and driver.Valuer.
type Amount big.Int

// PgType is the propagation type of an object.
type PgType string

const (
	// PgTpCanonical is an offer owned by this mint.
	PgTpCanonical PgType = "canonical"
	// PgTpPropagated is an offer propagated to this mint.
	PgTpPropagated PgType = "propagated"
)

// OfStatus is the status of an offer.
type OfStatus string

const (
	// OfStActive is used to mark an offer as active.
	OfStActive OfStatus = "active"
	// OfStClosed is used to mark an offer as closed.
	OfStClosed OfStatus = "closed"
	// OfStConsumed is used to mark an offer as consumed.
	OfStConsumed OfStatus = "consumed"
)

// OfPath is an offer path
type OfPath []string

// TxStatus is the status of a transaction, operation or crossing.
type TxStatus string

const (
	// TxStReserved is used to mark an action (operation or crossing) as
	// reserved.
	TxStReserved TxStatus = "reserved"
	// TxStSettled is used to mark an action (operation or crossing) as
	// settled.
	TxStSettled TxStatus = "settled"
	// TxStCanceled is used to mark an action (operation or crossing) as
	// canceled.
	TxStCanceled TxStatus = "canceled"
)

// Value implements driver.Valuer.
func (b Amount) Value() (value driver.Value, err error) {
	return (*big.Int)(&b).String(), nil
}

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
func (t PgType) Value() (value driver.Value, err error) {
	return string(t), nil
}

// Scan implements sql.Scanner.
func (t *PgType) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*t = PgType(src)
	case string:
		*t = PgType(src)
	default:
		return errors.Newf(
			"Incompatible type for PgType with value: %q", src)
	}

	return nil
}

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

// Value implements driver.Valuer.
func (p OfPath) Value() (value driver.Value, err error) {
	return strings.Join([]string(p), "/"), nil
}

// Scan implements sql.Scanner.
func (p *OfPath) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*p = strings.Split(string(src), "/")
	case string:
		*p = strings.Split(src, "/")
	default:
		return errors.Newf("Incompatible type for OfPath with value: %q", src)
	}

	return nil
}

// Value implements driver.Valuer.
func (s TxStatus) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *TxStatus) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = TxStatus(src)
	case string:
		*s = TxStatus(src)
	default:
		return errors.Newf(
			"Incompatible status for TxStatus with value: %q", src)
	}

	return nil
}
