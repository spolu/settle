// OWNER stan

package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
)

var defaultHTTPClient = (*http.Client)(nil)

// getDefaultHTTPClient returns the default HTTP client to use (to avoid
// re-instantiating one for each request)
func getDefaultHTTPClient(
	ctx context.Context,
) *http.Client {
	if defaultHTTPClient == nil {
		switch env.Get(ctx).Environment {
		case env.Production:
			defaultHTTPClient = &http.Client{}
		case env.QA:
			// In QA we don't check TLS certificates for ease of setup (see
			// GetSelfSignedQACertificate).
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			defaultHTTPClient = &http.Client{Transport: tr}
		}
	}
	return defaultHTTPClient
}

// Mint represents a mint
type Mint struct {
	Host        string
	Credentials *Credentials
}

// MintFromContextCredentials returns a mint object from the credentials stored
// in the current context.
func MintFromContextCredentials(
	ctx context.Context,
) (*Mint, error) {
	c := GetCredentials(ctx)
	if c == nil {
		return nil, errors.Trace(
			errors.Newf("Not logged in (see `settle login`)"))
	}
	return &Mint{
		Host:        c.Host,
		Credentials: c,
	}, nil
}

// Post performs a POST request to the mint.
func (m *Mint) Post(
	ctx context.Context,
	path string,
	query url.Values,
	params url.Values,
) (*int, *svc.Resp, error) {
	req, err := http.NewRequest("POST",
		mint.FullMintURL(ctx, m.Host, path, query).String(),
		strings.NewReader(params.Encode()))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	req.Header.Add("Mint-Protocol-Version", mint.ProtocolVersion)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m.Credentials != nil {
		req.SetBasicAuth(m.Credentials.Username, m.Credentials.Password)
	}

	r, err := getDefaultHTTPClient(ctx).Do(req)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, nil, errors.Trace(err)
	}

	return &r.StatusCode, &raw, nil
}

// Get performs a GET request to the mint.
func (m *Mint) Get(
	ctx context.Context,
	path string,
	query url.Values,
) (*int, *svc.Resp, error) {
	req, err := http.NewRequest("GET",
		mint.FullMintURL(ctx, m.Host, path, query).String(), nil)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	req.Header.Add("Mint-Protocol-Version", mint.ProtocolVersion)
	if m.Credentials != nil {
		req.SetBasicAuth(m.Credentials.Username, m.Credentials.Password)
	}

	r, err := getDefaultHTTPClient(ctx).Do(req)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, nil, errors.Trace(err)
	}

	return &r.StatusCode, &raw, nil
}
