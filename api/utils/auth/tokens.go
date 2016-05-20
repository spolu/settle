package auth

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/token"
	"github.com/stellar/go-stellar-base/keypair"
	"golang.org/x/net/context"
)

const (
	// TokenExpiry is the default expiry time for a token
	TokenExpiry = 5 * time.Minute
)

// MintToken generates a new valid token and returns it as a string along with
// its created date.
func MintToken(
	ctx context.Context,
	keypair *keypair.Full,
) (*string, *time.Time, error) {
	created := time.Now()
	token := fmt.Sprintf("%d", time.Now().UnixNano()/(1000*1000)) +
		"_" + token.RandStr()

	sign, err := keypair.Sign([]byte(token))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	token += ":" + hex.EncodeToString([]byte(sign))

	return &token, &created, nil
}

// CheckToken checks the validity of a token, returning user errors when a
// token is not valid.
func CheckToken(
	ctx context.Context,
	token string,
	keypair *keypair.Full,
) error {
	invalidTokenErr := func(err error, adj string) error {
		return errors.NewUserError(err,
			400,
			fmt.Sprintf("%s_token", adj),
			fmt.Sprintf("The token you used is %s: %s", adj, token),
		)
	}

	split := strings.Split(token, ":")
	if len(split) != 2 {
		return errors.Trace(invalidTokenErr(nil, "invalid"))
	}
	payload := split[0]
	check := split[1]

	split = strings.Split(payload, "_")
	if len(split) != 2 {
		return errors.Trace(invalidTokenErr(nil, "invalid"))
	}
	created, err := strconv.ParseInt(split[0], 10, 64)
	if err != nil {
		return errors.Trace(invalidTokenErr(err, "invalid"))
	}

	then := time.Unix(created/1000, 0)
	if time.Now().Sub(then) > TokenExpiry {
		return errors.Trace(invalidTokenErr(nil, "expired"))
	}

	sign, err := keypair.Sign([]byte(payload))
	if err != nil {
		return errors.Trace(err) // 500
	}
	if check != hex.EncodeToString([]byte(sign)) {
		return errors.Trace(invalidTokenErr(nil, "invalid"))
	}

	return nil
}
