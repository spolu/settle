package svc

import "fmt"

// ErrProtocolExtraction is returned when we fail to extract a protocol from a
// request or fail to parse its JSON.
type ErrProtocolExtraction struct {
	Protocol string
}

func (e ErrProtocolExtraction) Error() string {
	return fmt.Sprintf(
		"Request protocol extraction failed: %s", e.Protocol)
}
