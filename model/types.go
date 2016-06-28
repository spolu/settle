package model

import (
	"database/sql/driver"
	"fmt"
	"math/big"
	"reflect"

	"github.com/spolu/settle/lib/errors"
)

// BigInt extends big.Int to implement sql.Scanner and driver.Valuer.
type BigInt big.Int

// Scan implements sql.Scanner.
func (b *BigInt) Scan(value interface{}) error {
	switch value := value.(type) {
	case int64:
		(*big.Int)(b).SetInt64(value)
	case []byte:
		if _, err := fmt.Sscan(string(value), b); err != nil {
			return errors.Trace(err)
		}
	case string:
		if _, err := fmt.Sscan(value, b); err != nil {
			return errors.Trace(err)
		}
	default:
		return errors.Newf("Cannot scan %+v for: %q",
			reflect.TypeOf(value), value)
	}

	return nil
}

// Value implements driver.Valuer
func (b BigInt) Value() (value driver.Value, err error) {
	return (*big.Int)(&b).String(), nil
}
