package model

import (
	"database/sql/driver"
	"math/big"

	"github.com/spolu/settle/lib/errors"
)

// BigInt extends big.Int to implement sql.Scanner and driver.Valuer.
type BigInt big.Int

// Scan implements sql.Scanner.
func (b *BigInt) Scan(src interface{}) error {
	switch src := src.(type) {
	case int64:
		(*big.Int)(b).SetInt64(src)
	case []byte:
		if _, success := (*big.Int)(b).SetString(string(src), 10); !success {
			return errors.Newf("Impossible to set BigInt with string: %q", src)
		}
	case string:
		if _, success := (*big.Int)(b).SetString(src, 10); !success {
			return errors.Newf("Impossible to set BigInt with string: %q", src)
		}
	default:
		return errors.Newf("Incompatible type for BigInt with value: %q", src)
	}

	return nil
}

// Value implements driver.Valuer
func (b BigInt) Value() (value driver.Value, err error) {
	return (*big.Int)(&b).String(), nil
}
