package client

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/spolu/settle/lib/env"
)

var defaultClient = (*http.Client)(nil)

// getDefaultHTTPClient returns the default HTTP client to use (to avoid
// re-instantiating one for each request)
func Default(
	ctx context.Context,
) *http.Client {
	if defaultClient == nil {
		switch env.Get(ctx).Environment {
		case env.Production:
			defaultClient = &http.Client{}
		case env.QA:
			// In QA we don't check TLS certificates for ease of setup (see
			// GetSelfSignedQACertificate).
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			defaultClient = &http.Client{Transport: tr}
		}
	}
	return defaultClient
}
