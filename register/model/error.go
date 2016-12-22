package model

import "fmt"

// ErrUniqueConstraintViolation is returned when a object insertion violates a
// unique constraint.
type ErrUniqueConstraintViolation struct {
	Err error
}

func (e ErrUniqueConstraintViolation) Error() string {
	return fmt.Sprintf(
		"Unique constraint violation in %s", e.Err.Error())
}
