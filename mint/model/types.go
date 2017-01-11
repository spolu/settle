package model

import (
	"database/sql/driver"
	"math/big"
	"strings"

	"github.com/spolu/settle/lib/errors"
)

// Amount extends big.Int to implement sql.Scanner and driver.Valuer.
type Amount big.Int

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

// OfPath is an offer path and implemets sql.Scanner and dirver.Valuer for easy
// serialization.
type OfPath []string

// Value implements driver.Valuer.
func (p OfPath) Value() (value driver.Value, err error) {
	return strings.Join([]string(p), "/"), nil
}

// Scan implements sql.Scanner.
func (p *OfPath) Scan(src interface{}) error {
	s := ""
	switch src := src.(type) {
	case []byte:
		s = string(src)
	case string:
		s = src
	default:
		return errors.Newf("Incompatible type for OfPath with value: %q", src)
	}
	if len(s) == 0 {
		*p = []string{}
	} else {
		*p = strings.Split(s, "/")
	}

	return nil
}
