package mint

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"

	"golang.org/x/net/context"
)

// Client expose an interface to perform queries on remote mints.
type Client struct {
}

// AssetNameRegexp is used to validate and parse asset names.
var AssetNameRegexp = regexp.MustCompile(
	"^([a-zA-Z0-9\\-_.]{1,256})(\\+[a-zA-Z0-9\\-_.]+){0,1}@([a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+):([A-Z0-9\\-]{1,64})\\.([0-9]{1,2})$")

// AddressRegexp is used to validate and parse issuer names.
var AddressRegexp = regexp.MustCompile(
	"^([a-zA-Z0-9\\-_.]{1,256})(\\+[a-zA-Z0-9\\-_.]+){0,1}@([a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+)$")

// AssetResourceFromName parses an asset fully qualified name into an
// AssetResource object (without id or created date). Livemode is infered by
// the current context.
func (c *Client) AssetResourceFromName(
	ctx context.Context,
	name string,
) (*AssetResource, error) {
	m := AssetNameRegexp.FindStringSubmatch(name)
	if len(m) == 0 {
		return nil, errors.Trace(errors.Newf("Invalid asset name: %s", name))
	}
	s, err := strconv.ParseInt(m[5], 10, 8)
	if err != nil {
		return nil, errors.Trace(errors.Newf("Invalid asset name: %s", name))
	}

	return &AssetResource{
		Livemode: livemode.Get(ctx),
		Name:     name,
		Issuer:   fmt.Sprintf("%s@%s", m[1], m[3]),
		Code:     m[4],
		Scale:    int8(s),
	}, nil
}

// UsernameAndMintHostFromAddress extracts the username and mint host from a
// fully qualified address.
func (c *Client) UsernameAndMintHostFromAddress(
	ctx context.Context,
	issuer string,
) (string, string, error) {
	m := AddressRegexp.FindStringSubmatch(issuer)
	if len(m) == 0 {
		return "", "", errors.Trace(errors.Newf(
			"Invalid issuer address: %s", issuer))
	}

	return m[1], m[3], nil
}
