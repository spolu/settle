package mint

import "fmt"

// ErrMintClient is returned by the client when an proper error is returned by
// the mint it interacted with.
type ErrMintClient struct {
	StatusCode int
	ErrCode    string
	ErrMessage string
}

func (e ErrMintClient) Error() string {
	return fmt.Sprintf(
		"[%d] (%s) %s", e.StatusCode, e.ErrCode, e.ErrMessage)
}
