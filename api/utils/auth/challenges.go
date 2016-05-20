package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/token"
	"github.com/stellar/go-stellar-base/keypair"
	"golang.org/x/net/context"
)

// MintChallenge generates a new valid challenge and returns it as a string
// along with its created date.
func MintChallenge(
	ctx context.Context,
	kp *keypair.Full,
) (*string, *time.Time, error) {
	created := time.Now()
	challenge := token.RandStr()

	sign, err := kp.Sign([]byte(challenge))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	challenge += ":" + base64.StdEncoding.EncodeToString([]byte(sign))

	return &challenge, &created, nil
}

// CheckChallenge checks the validity of a challenge, returning user errors
// when a challenge is not valid.
func CheckChallenge(
	ctx context.Context,
	challenge string,
	kp *keypair.Full,
) error {
	invalidChallengeErr := func(err error) error {
		return errors.NewUserError(err,
			400, "challenge_invalid",
			"The challenge you provided is invalid. It was probably altered "+
				"since it was retrieved from the API.",
		)
	}

	split := strings.Split(challenge, ":")
	if len(split) != 2 {
		return errors.Trace(invalidChallengeErr(nil))
	}
	payload := split[0]
	check := split[1]

	sign, err := kp.Sign([]byte(payload))
	if err != nil {
		return errors.Trace(err) // 500
	}
	if check != base64.StdEncoding.EncodeToString([]byte(sign)) {
		return errors.Trace(invalidChallengeErr(nil))
	}

	return nil
}

// VerifyChallenge verify a challenge signature, returning user errors when the
// signature is invalid.
func VerifyChallenge(
	ctx context.Context,
	challenge string,
	address string,
	signature string,
) error {
	kp, err := keypair.Parse(address)
	if err != nil {
		return errors.NewUserError(err,
			400, "invalid_address",
			fmt.Sprintf(
				"The address you provided as authentication is invalid: %s.",
				address,
			),
		)
	}

	bytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return errors.NewUserError(err,
			400, "invalid_signature",
			"The signature passed could not be decoded.",
		)
	}

	err = kp.Verify([]byte(challenge), bytes)
	if err != nil {
		return errors.NewUserError(err,
			400, "invalid_signature",
			"The verification of the challenge signature failed using the "+
				"address you provided.",
		)
	}

	return nil
}
