package mint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/spolu/settle/lib/client"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/svc"
)

// Client expose an interface to perform queries on remote mints.
type Client struct {
	httpClient *http.Client
}

// Init initializes the mint client.
func (c *Client) Init(
	ctx context.Context,
) error {
	c.httpClient = client.Default(ctx)
	return nil
}

// DefaultPort is the mint default port by environment.
var DefaultPort = map[env.Environment]int64{
	env.Production: 2406,
	env.QA:         2407,
}

// DefaultScheme is the mint default scheme by environment.
var DefaultScheme = map[env.Environment]string{
	env.Production: "https",
	env.QA:         "http",
}

// Possible address: von.neumann@ias.edu:8989
var addressRegexpStr = "([a-zA-Z0-9-_.]{1,256})(\\+[a-zA-Z0-9-_.]+){0,1}@" +
	"([a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+(:[0-9]{1,5}){0,1})"

// AssetNameRegexp is used to validate and parse asset names.
var AssetNameRegexp = regexp.MustCompile(
	"^" + addressRegexpStr + "\\[([A-Z0-9-]{1,64})\\.([0-9]{1,2})\\]" + "$",
)

// AddressRegexp is used to validate and parse issuer names.
var AddressRegexp = regexp.MustCompile(
	"^" + addressRegexpStr + "$",
)

// IDRegexp is used to validate a full id including issuer and token.
var IDRegexp = regexp.MustCompile(
	"^(.+)\\[([a-z]+_[a-zA-Z0-9]+)\\]$",
)

// AssetResourceFromName parses an asset fully qualified name into an
// AssetResource object (without id or created date, owner is normalized).
func AssetResourceFromName(
	ctx context.Context,
	name string,
) (*AssetResource, error) {
	m := AssetNameRegexp.FindStringSubmatch(name)
	if len(m) == 0 {
		return nil, errors.Trace(errors.Newf("Invalid asset name: %s", name))
	}
	s, err := strconv.ParseInt(m[6], 10, 8)
	if err != nil || s < 0 || s > 24 {
		return nil, errors.Trace(errors.Newf("Invalid asset scale: %s", m[6]))
	}

	return &AssetResource{
		Owner: fmt.Sprintf("%s@%s", m[1], m[3]),
		Name:  name,
		Code:  m[5],
		Scale: int8(s),
	}, nil
}

// AssetResourcesFromPair parses a pair into an array of AssetResources
// (without id or created date).
func AssetResourcesFromPair(
	ctx context.Context,
	pair string,
) ([]AssetResource, error) {
	ss := strings.Split(pair, "/")
	if len(ss) != 2 {
		return nil, errors.Trace(errors.Newf("Invalid asset pair: %s", pair))
	}
	base, err := AssetResourceFromName(ctx, ss[0])
	if err != nil {
		return nil, errors.Trace(err)
	}
	quote, err := AssetResourceFromName(ctx, ss[1])
	if err != nil {
		return nil, errors.Trace(err)
	}
	return []AssetResource{*base, *quote}, nil
}

// UsernameAndMintHostFromAddress extracts the username and mint host from a
// fully qualified address.
func UsernameAndMintHostFromAddress(
	ctx context.Context,
	address string,
) (string, string, error) {
	m := AddressRegexp.FindStringSubmatch(address)
	if len(m) == 0 {
		return "", "", errors.Trace(errors.Newf(
			"Invalid address: %s", address))
	}

	return m[1], m[3], nil
}

// NormalizedAddress returns the address trimmed from the `+...@` part.
func NormalizedAddress(
	ctx context.Context,
	address string,
) (string, error) {
	m := AddressRegexp.FindStringSubmatch(address)
	if len(m) == 0 {
		return "", errors.Trace(errors.Newf("Invalid address: %s", address))
	}

	return fmt.Sprintf("%s@%s", m[1], m[3]), nil
}

// NormalizedOwnerAndTokenFromID returns a normalized address and token from
// an id.
func NormalizedOwnerAndTokenFromID(
	ctx context.Context,
	id string,
) (string, string, error) {
	m := IDRegexp.FindStringSubmatch(id)
	if len(m) == 0 {
		return "", "", errors.Trace(errors.Newf("Invalid id: %s", id))
	}
	owner, err := NormalizedAddress(ctx, m[1])
	if err != nil {
		return "", "", errors.Trace(err)
	}
	return owner, m[2], nil
}

// FullMintURL constructs a fully qualified URL to contact a mint defaulting to
// the correct scheme and port based on the current environment.
func FullMintURL(
	ctx context.Context,
	host string,
	path string,
	query url.Values,
) *url.URL {
	if len(strings.Split(host, ":")) == 1 {
		host += fmt.Sprintf(":%d", DefaultPort[env.Get(ctx).Environment])
	}
	url := url.URL{
		Scheme:   DefaultScheme[env.Get(ctx).Environment],
		Host:     host,
		Path:     path,
		RawQuery: query.Encode(),
	}
	return &url
}

// RetrieveBalance retrieves an balance given its ID by extracting the mint and
// retrieving it from there.
func (c *Client) RetrieveBalance(
	ctx context.Context,
	id string,
) (*BalanceResource, error) {
	owner, _, err := NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	_, host, err := UsernameAndMintHostFromAddress(ctx, owner)
	if err != nil {
		return nil, errors.Trace(err)
	}

	req, err := http.NewRequest("GET",
		FullMintURL(ctx,
			host, fmt.Sprintf("/balances/%s", id), url.Values{}).String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var balance BalanceResource
	if err := raw.Extract("balance", &balance); err != nil {
		return nil, errors.Trace(err)
	}

	return &balance, nil
}

// RetrieveOffer retrieves an offer given its ID by extracting the mint and
// retrieving it from there.
func (c *Client) RetrieveOffer(
	ctx context.Context,
	id string,
) (*OfferResource, error) {
	owner, _, err := NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	_, host, err := UsernameAndMintHostFromAddress(ctx, owner)
	if err != nil {
		return nil, errors.Trace(err)
	}

	req, err := http.NewRequest("GET",
		FullMintURL(ctx,
			host, fmt.Sprintf("/offers/%s", id), url.Values{}).String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var offer OfferResource
	if err := raw.Extract("offer", &offer); err != nil {
		return nil, errors.Trace(err)
	}

	return &offer, nil
}

// RetrieveOperation retrieves an operation given its ID by extracting the mint
// and retrieving it from there.
func (c *Client) RetrieveOperation(
	ctx context.Context,
	id string,
) (*OperationResource, error) {
	owner, _, err := NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	_, host, err := UsernameAndMintHostFromAddress(ctx, owner)
	if err != nil {
		return nil, errors.Trace(err)
	}

	req, err := http.NewRequest("GET",
		FullMintURL(ctx,
			host, fmt.Sprintf("/operations/%s", id), url.Values{}).String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var operation OperationResource
	if err := raw.Extract("operation", &operation); err != nil {
		return nil, errors.Trace(err)
	}

	return &operation, nil
}

// RetrieveTransaction retrieves a transaction given its ID by extracting the
// mint and retrieving it from there. If host is specified, it attempts to
// retrrieve the transaction from this host instead of the canonical host.
func (c *Client) RetrieveTransaction(
	ctx context.Context,
	id string,
	mint *string,
) (*TransactionResource, error) {
	if mint == nil {
		owner, _, err := NormalizedOwnerAndTokenFromID(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		_, host, err := UsernameAndMintHostFromAddress(ctx, owner)
		if err != nil {
			return nil, errors.Trace(err)
		}
		mint = &host
	}

	req, err := http.NewRequest("GET",
		FullMintURL(ctx,
			*mint, fmt.Sprintf("/transactions/%s", id), url.Values{}).String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var transaction TransactionResource
	if err := raw.Extract("transaction", &transaction); err != nil {
		return nil, errors.Trace(err)
	}

	return &transaction, nil
}

// PropagateBalance propagates an balance to the specified mint.
func (c *Client) PropagateBalance(
	ctx context.Context,
	id string,
	mint string,
) (*BalanceResource, error) {
	req, err := http.NewRequest("POST",
		FullMintURL(ctx, mint,
			fmt.Sprintf("/balances/%s", id), url.Values{}).String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var balance BalanceResource
	if err := raw.Extract("balance", &balance); err != nil {
		return nil, errors.Trace(err)
	}

	return &balance, nil
}

// PropagateOffer propagates an offer to the specified mint.
func (c *Client) PropagateOffer(
	ctx context.Context,
	id string,
	mint string,
) (*OfferResource, error) {
	req, err := http.NewRequest("POST",
		FullMintURL(ctx, mint,
			fmt.Sprintf("/offers/%s", id), url.Values{}).String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var offer OfferResource
	if err := raw.Extract("offer", &offer); err != nil {
		return nil, errors.Trace(err)
	}

	return &offer, nil
}

// PropagateOperation propagates an operation to the specified mint.
func (c *Client) PropagateOperation(
	ctx context.Context,
	id string,
	mint string,
) (*OperationResource, error) {
	req, err := http.NewRequest("POST",
		FullMintURL(ctx, mint,
			fmt.Sprintf("/operations/%s", id), url.Values{}).String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var operation OperationResource
	if err := raw.Extract("operation", &operation); err != nil {
		return nil, errors.Trace(err)
	}

	return &operation, nil
}

// PropagateTransaction propagates a transaction to the specified mint.
func (c *Client) PropagateTransaction(
	ctx context.Context,
	id string,
	hop int8,
	mint string,
) (*TransactionResource, error) {
	req, err := http.NewRequest("POST",
		FullMintURL(ctx, mint,
			fmt.Sprintf("/transactions/%s", id), url.Values{}).String(),
		strings.NewReader(url.Values{
			"hop": {fmt.Sprintf("%d", hop)},
		}.Encode()))
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var transaction TransactionResource
	if err := raw.Extract("transaction", &transaction); err != nil {
		return nil, errors.Trace(err)
	}

	return &transaction, nil
}

// SettleTransaction settles a transaction. If hop, secret, mint are specified
// it settles on the specified mint using the specified hop and secret,
// otherwise it attempts to settle on the canonical mint.
func (c *Client) SettleTransaction(
	ctx context.Context,
	id string,
	hop *int8,
	secret *string,
	mint *string,
) (*TransactionResource, error) {
	if mint == nil {
		owner, _, err := NormalizedOwnerAndTokenFromID(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		_, host, err := UsernameAndMintHostFromAddress(ctx, owner)
		if err != nil {
			return nil, errors.Trace(err)
		}
		mint = &host
	}

	body := url.Values{}
	if hop != nil {
		body["hop"] = []string{fmt.Sprintf("%d", *hop)}
	}
	if secret != nil {
		body["secret"] = []string{*secret}
	}

	req, err := http.NewRequest("POST",
		FullMintURL(ctx, *mint,
			fmt.Sprintf("/transactions/%s/settle", id), url.Values{}).String(),
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var transaction TransactionResource
	if err := raw.Extract("transaction", &transaction); err != nil {
		return nil, errors.Trace(err)
	}

	return &transaction, nil
}

// CancelTransaction propagates the cancelation of a transaction on the
// specified mint for the specified hop.
func (c *Client) CancelTransaction(
	ctx context.Context,
	id string,
	hop int8,
	mint string,
) (*TransactionResource, error) {
	body := url.Values{
		"hop": []string{fmt.Sprintf("%d", hop)},
	}
	req, err := http.NewRequest("POST",
		FullMintURL(ctx, mint,
			fmt.Sprintf("/transactions/%s/cancel", id), url.Values{}).String(),
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Mint-Protocol-Version", ProtocolVersion)
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(ErrMintClient{
			r.StatusCode, e.ErrCode, e.ErrMessage,
		})
	}

	var transaction TransactionResource
	if err := raw.Extract("transaction", &transaction); err != nil {
		return nil, errors.Trace(err)
	}

	return &transaction, nil
}
