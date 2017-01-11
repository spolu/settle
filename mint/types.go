package mint

import (
	"database/sql/driver"

	"github.com/spolu/settle/lib/errors"
)

// SQL interface for custom types.

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

// Value implements driver.Valuer.
func (s TkStatus) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *TkStatus) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = TkStatus(src)
	case string:
		*s = TkStatus(src)
	default:
		return errors.Newf(
			"Incompatible status for TkStatus with value: %q", src)
	}

	return nil
}

// Value implements driver.Valuer.
func (s TkName) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *TkName) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = TkName(src)
	case string:
		*s = TkName(src)
	default:
		return errors.Newf(
			"Incompatible status for TkName with value: %q", src)
	}

	return nil
}
