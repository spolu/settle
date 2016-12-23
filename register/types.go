// OWNER: stan

package register

import (
	"database/sql/driver"

	"github.com/spolu/settle/lib/errors"
)

// Value implements driver.Valuer.
func (s UsrStatus) Value() (value driver.Value, err error) {
	return string(s), nil
}

// Scan implements sql.Scanner.
func (s *UsrStatus) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = UsrStatus(src)
	case string:
		*s = UsrStatus(src)
	default:
		return errors.Newf(
			"Incompatible status for UsrStatus with value: %q", src)
	}

	return nil
}
