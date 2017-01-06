package functional

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"goji.io/pat"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/endpoint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupCreateTransactionFailure(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser, []mint.AssetResource, []mint.OfferResource) {
	m := []*test.Mint{
		test.CreateMint(t),
		test.CreateMint(t),
		test.CreateMint(t),
	}
	u := []*test.MintUser{
		m[0].CreateUser(t),
		m[1].CreateUser(t),
		m[2].CreateUser(t),
	}
	a := []mint.AssetResource{
		u[0].CreateAsset(t, "USD", 2),
		u[1].CreateAsset(t, "USD", 2),
		u[2].CreateAsset(t, "USD", 2),
	}

	o := []mint.OfferResource{
		u[0].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address),
			"100/100", big.NewInt(100)),
		u[1].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[1].Address, u[0].Address),
			"100/100", big.NewInt(100)),
		u[2].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[2].Address, u[1].Address),
			"100/100", big.NewInt(100)),
	}

	return m, u, a, o
}

func tearDownCreateTransactionFailure(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestCreateTransactionFailureMultiCreation(
	t *testing.T,
) {
	t.Parallel()
	m, u, _, o := setupCreateTransactionFailure(t)
	defer tearDownCreateTransactionFailure(t, m)

	repostDone := false
	// Intercept transaction propagation and attempt to repost
	m[1].Mux.Use(func(inner http.Handler) http.Handler {
		mw := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			pattern := regexp.MustCompile("^/transactions/[a-zA-Z0-9_\\+:@\\.\\[\\]]+$")

			if r.Method == "POST" &&
				pattern.MatchString(r.URL.Path) &&
				!repostDone {
				repostDone = true

				id, _, _, err := endpoint.ValidateID(ctx,
					pat.Param(r, "transaction"))
				assert.Nil(t, err)

				fmt.Printf("\n ---> %s REPOST ATTACK\n\n", r.URL.Path)

				m[2].Post(t,
					nil,
					fmt.Sprintf("/transactions/%s", *id),
					url.Values{
						"hop": {"2"},
					})

				inner.ServeHTTP(w, r)
			} else {
				fmt.Printf(" ---> %s SKIPn\n", r.URL.Path)
				inner.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(mw)
	})

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]": {
				o[1].ID,
				o[2].ID,
			},
		})

	var tx0 mint.TransactionResource
	err := raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)

	status, raw = u[1].Get(t, fmt.Sprintf("/offers/%s", o[1].ID))

	var of1 mint.OfferResource
	err = raw.Extract("offer", &of1)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)

	// We should not have crossed the offer twice.
	assert.Equal(t, big.NewInt(90), of1.Remainder)
}
