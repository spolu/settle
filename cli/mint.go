// OWNER stan

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
)

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
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m.Credentials != nil {
		req.SetBasicAuth(m.Credentials.Username, m.Credentials.Password)
	}

	r, err := http.DefaultClient.Do(req)
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
	if m.Credentials != nil {
		req.SetBasicAuth(m.Credentials.Username, m.Credentials.Password)
	}

	r, err := http.DefaultClient.Do(req)
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
