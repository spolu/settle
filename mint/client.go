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

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
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
	c.httpClient = &http.Client{}
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
var addressRegexpStr = "([a-zA-Z0-9\\-_.]{1,256})(\\+[a-zA-Z0-9\\-_.]+){0,1}@" +
	"([a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+(:[0-9]{1,5}){0,1})"

// AssetNameRegexp is used to validate and parse asset names.
var AssetNameRegexp = regexp.MustCompile(
	"^" + addressRegexpStr + "\\[([A-Z0-9\\-]{1,64})\\.([0-9]{1,2})\\]" + "$",
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
// AssetResource object (without id or created date). Livemode is infered by
// the current context.
func AssetResourceFromName(
	ctx context.Context,
	name string,
) (*AssetResource, error) {
	m := AssetNameRegexp.FindStringSubmatch(name)
	if len(m) == 0 {
		return nil, errors.Trace(errors.Newf("Invalid asset name: %s", name))
	}
	s, err := strconv.ParseInt(m[6], 10, 8)
	if err != nil {
		return nil, errors.Trace(errors.Newf("Invalid asset name: %s", name))
	}

	return &AssetResource{
		Livemode: livemode.Get(ctx),
		Name:     name,
		Issuer:   fmt.Sprintf("%s@%s", m[1], m[3]),
		Code:     m[5],
		Scale:    int8(s),
	}, nil
}

// AssetResourcesFromPair parses a pair into an array of AssetResources
// (without id or created date). Livemode is infered by the current context.
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

// NormalizedAddressAndTokenFromID returns a normalized address and token from
// an id.
func NormalizedAddressAndTokenFromID(
	ctx context.Context,
	id string,
) (string, string, error) {
	m := IDRegexp.FindStringSubmatch(id)
	if len(m) == 0 {
		return "", "", errors.Trace(errors.Newf("Invalid id: %s", id))
	}
	address, err := NormalizedAddress(ctx, m[1])
	if err != nil {
		return "", "", errors.Trace(err)
	}
	return address, m[2], nil
}

// FullMintURL constructs a fully qualified URL to contact a mint defaulting to
// the correct scheme and port based on the current environment.
func FullMintURL(
	ctx context.Context,
	host string,
	path string,
) *url.URL {
	if len(strings.Split(host, ":")) == 1 {
		host += fmt.Sprintf(":%d", DefaultPort[env.Get(ctx).Environment])
	}
	url := url.URL{
		Scheme: DefaultScheme[env.Get(ctx).Environment],
		Host:   host,
		Path:   path,
	}
	return &url
}

// RetrieveOffer retrieves an offer given its ID by extracting the mint and
// retrieving it from there.
func (c *Client) RetrieveOffer(
	ctx context.Context,
	id string,
) (*OfferResource, error) {
	address, _, err := NormalizedAddressAndTokenFromID(ctx, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	_, host, err := UsernameAndMintHostFromAddress(ctx, address)
	if err != nil {
		return nil, errors.Trace(err)
	}

	r, err := c.httpClient.Get(
		FullMintURL(ctx, host, fmt.Sprintf("/offers/%s", id)).String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	var offer OfferResource
	if err := raw.Extract("ofer", &offer); err != nil {
		return nil, errors.Trace(err)
	}

	return &offer, nil
}
