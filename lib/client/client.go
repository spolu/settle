package client

import (
	"context"
	"net/http"
)

var defaultClient = (*http.Client)(nil)

// Default returns the default HTTP client to use (to avoid re-instantiating
// one for each request)
func Default(
	ctx context.Context,
) *http.Client {
	if defaultClient == nil {
		defaultClient = &http.Client{}
	}
	return defaultClient
}
