package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/spolu/settl/api/utils/auth"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/format"
	"github.com/spolu/settl/util/respond"
	"github.com/spolu/settl/util/svc"

	"golang.org/x/net/context"
)

const (
	// DefaultRetrieveTokensCount is the default number of tokens returned by
	// the API if the count attribute is not specified.
	DefaultRetrieveTokensCount = uint64(10)
	// MaxRetrieveTokensCount is the maximium number of tokens that can be
	// retrieved.
	MaxRetrieveTokensCount = uint64(10)
)

type controller struct{}

func (c *controller) RetrieveTokens(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	count := DefaultRetrieveTokensCount
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

	tokens := []TokenResource{}
	for i := uint64(0); i < count; i++ {
		token, created, err := auth.MintToken(ctx, auth.RootLiveKeypair)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err)) // 500
			return

		}
		tokens = append(tokens, TokenResource{
			ID:      *token,
			Created: (*created).UnixNano() / (1000 * 1000),
			ExpiresAt: (*created).
				Add(auth.TokenExpiry).UnixNano() / (1000 * 1000),
		})
	}

	respond.Created(ctx, w, svc.Resp{
		"tokens": format.JSONPtr(tokens),
	})
}

func (c *controller) CreateUser(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
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
