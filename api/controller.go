package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/spolu/settl/api/util/auth"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/format"
	"github.com/spolu/settl/util/respond"
	"github.com/spolu/settl/util/svc"

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
					"invalid_count_attribute",
					fmt.Sprintf("The count attribute you passed is not valid "+
						"(should be a positive integer smaller than 100): %s",
						attr),
				)))
			return
		}
	}

	challenges := []ChallengeResource{}
	for i := uint64(0); i < count; i++ {
		challenge, created, err := auth.MintChallenge(ctx, auth.RootLiveKeypair)
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

var usernameRegexp = regexp.MustCompile("^[a-z0-9]+$")

func (c *controller) CreateUser(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {

	params := UserParams{
		Username:      r.PostFormValue("username"),
		EncryptedSeed: r.PostFormValue("encrypted_seed"),
	}

	if !usernameRegexp.MatchString(params.Username) {
		respond.Error(ctx, w, errors.NewUserError(nil,
			400, "username_invalid",
			"The username provided is invalid. Usernames must be "+
				"alphanumeric lowercased characters only.",
		))
		return
	}
}

func (c *controller) CreateStellarOperation(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}

func (c *controller) SubmitStellarOperation(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}
