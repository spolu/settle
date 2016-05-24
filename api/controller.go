package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/spolu/settl/api/lib/authentication"
	"github.com/spolu/settl/api/lib/livemode"
	"github.com/spolu/settl/lib/errors"
	"github.com/spolu/settl/lib/format"
	"github.com/spolu/settl/lib/respond"
	"github.com/spolu/settl/lib/svc"

	"golang.org/x/net/context"
)

const (
	// DefaultRetrieveChallengesCount is the default number of challenges
	// returned by the API if the count attribute is not specified.
	DefaultRetrieveChallengesCount = uint64(10)
	// MaxRetrieveChallengesCount is the maximium number of challenges that can
	// be retrieved.
	MaxRetrieveChallengesCount = uint64(10)
)

type controller struct{}

func (c *controller) RetrieveChallenges(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	count := DefaultRetrieveChallengesCount
	if attr := r.URL.Query().Get("count"); attr != "" {
		err := error(nil)
		count, err = strconv.ParseUint(attr, 10, 64)
		if err != nil || count >= 100 {
			respond.Error(ctx, w, errors.Trace(
				errors.NewUserError(err,
					400,
					"count_invalid",
					fmt.Sprintf("The count attribute you passed is not valid "+
						"(should be a positive integer smaller than 100): %s",
						attr),
				)))
			return
		}
	}

	challenges := []ChallengeResource{}
	for i := uint64(0); i < count; i++ {
		challenge, created, err :=
			authentication.MintChallenge(ctx, authentication.RootLiveKeypair)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return

		}
		challenges = append(challenges, ChallengeResource{
			Value:   *challenge,
			Created: (*created).UnixNano() / (1000 * 1000),
		})
	}

	respond.Success(ctx, w, svc.Resp{
		"challenges": format.JSONPtr(challenges),
	})
}

var usernameRegexp = regexp.MustCompile(
	"^[a-z0-9]+$")
var emailRegexp = regexp.MustCompile(
	"^[a-z0-9_\\.\\+\\-]+@[a-z0-9-]+\\.[a-z0-9-\\.]+$")
var emailVerifiers = map[bool][]string{
	true: []string{
		"GBTIKKWP5FOCMRSTJS46SCTWC6IKCHWDJMJMP6QLFGNYPRTCY63E5T3N",
	},
	false: []string{},
}

func (c *controller) CreateUser(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	params := UserParams{
		Livemode:           livemode.Get(ctx),
		Address:            authentication.Get(ctx).Address,
		Username:           r.PostFormValue("username"),
		EncryptedSeed:      r.PostFormValue("encrypted_seed"),
		Email:              strings.ToLower(r.PostFormValue("email")),
		EmailVerifier:      r.PostFormValue("email_verifier"),
		FundingTransaction: r.PostFormValue("funding_transaction"),
	}

	if !usernameRegexp.MatchString(params.Username) {
		respond.Error(ctx, w, errors.NewUserError(nil,
			400, "username_invalid",
			"The username provided is invalid. Usernames can use "+
				"alphanumeric lowercased characters only.",
		))
		return
	}
	if !emailRegexp.MatchString(params.Email) {
		respond.Error(ctx, w, errors.NewUserError(nil,
			400, "email_invalid",
			"The email provided appears to be invalid. While email "+
				"verification is a bit tricky, we really try to do our best.",
		))
		return
	}
	_, err := base64.StdEncoding.DecodeString(params.EncryptedSeed)
	if err != nil {
		respond.Error(ctx, w, errors.NewUserError(err,
			400, "encrypted_seed_invalid",
			"The encrypted seed appears to be invalid as it could not be "+
				"decoded using base64. The encrypted seed should be the XOR "+
				"of the raw seed and an scrypt output of the same length "+
				"using base64 standard encoding.",
		))
	}

	// - check that account with same address does not exist
	// - check that account with same username does not exist
	// - check email fact on specified emailVerifier
	// - check that the funding transaction hasn't been used yet
	// - check that the funding transaction has exactly 1 operation and funds
	//   the root key with > 100 XLM
}

func (c *controller) ConfirmUser(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}

func (c *controller) CreateNativeOperation(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}

func (c *controller) SubmitNativeOperation(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}
