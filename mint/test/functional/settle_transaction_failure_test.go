package functional

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/endpoint"
	"github.com/spolu/settle/mint/model"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
	"goji.io/pat"
)

func setupSettleTransactionFailure(
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

func tearDownSettleTransactionFailure(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestSettleTransactionFailurePrepostSettlementOnSuccessor(
	t *testing.T,
) {
	t.Parallel()
	m, u, a, o := setupSettleTransactionFailure(t)
	defer tearDownSettleTransactionFailure(t, m)

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

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)

	repostDone := false
	// Intercept transaction propagation and attempt to repost
	m[2].Mux.Use(func(inner http.Handler) http.Handler {
		mw := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			pattern := regexp.MustCompile("^/transactions/[a-zA-Z0-9_\\+:@\\.\\[\\]]+/settle$")

			if r.Method == "POST" &&
				pattern.MatchString(r.URL.Path) &&
				!repostDone {
				repostDone = true

				id, _, _, err := endpoint.ValidateID(ctx,
					pat.Param(r, "transaction"))
				assert.Nil(t, err)

				secret, err := endpoint.ValidateSecret(ctx,
					r.PostFormValue("secret"))
				assert.Nil(t, err)

				fmt.Printf("\n ---> %s PREPOST ATTACK\n\n", r.URL.Path)

				m[1].Post(t,
					nil,
					fmt.Sprintf("/transactions/%s/settle", *id),
					url.Values{
						"hop":    {"1"},
						"secret": {*secret},
					})

				inner.ServeHTTP(w, r)
			} else {
				fmt.Printf(" ---> %s SKIP\n", r.URL.Path)
				inner.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(mw)
	})

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	var tx0 mint.TransactionResource
	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)

	// Check balance on m[0]
	balance, err := model.LoadCanonicalBalanceByAssetHolder(m[0].Ctx,
		a[0].Name, u[1].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(10), (*big.Int)(&balance.Value))

	// Check balance on m[1]
	balance, err = model.LoadCanonicalBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[2].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(10), (*big.Int)(&balance.Value))
}

func TestSettleTransactionFailurePrepostSettlementOnWrongHost(
	t *testing.T,
) {
	t.Parallel()
	m, u, a, o := setupSettleTransactionFailure(t)
	defer tearDownSettleTransactionFailure(t, m)

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

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)

	repostDone := false
	// Intercept transaction propagation and attempt to repost
	m[2].Mux.Use(func(inner http.Handler) http.Handler {
		mw := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			pattern := regexp.MustCompile("^/transactions/[a-zA-Z0-9_\\+:@\\.\\[\\]]+/settle$")

			if r.Method == "POST" &&
				pattern.MatchString(r.URL.Path) &&
				!repostDone {
				repostDone = true

				id, _, _, err := endpoint.ValidateID(ctx,
					pat.Param(r, "transaction"))
				assert.Nil(t, err)

				secret, err := endpoint.ValidateSecret(ctx,
					r.PostFormValue("secret"))
				assert.Nil(t, err)

				fmt.Printf("\n ---> %s PREPOST ATTACK\n\n", r.URL.Path)

				m[1].Post(t,
					nil,
					fmt.Sprintf("/transactions/%s/settle", *id),
					url.Values{
						"hop":    {"0"},
						"secret": {*secret},
					})

				inner.ServeHTTP(w, r)
			} else {
				fmt.Printf(" ---> %s SKIP\n", r.URL.Path)
				inner.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(mw)
	})

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	var tx0 mint.TransactionResource
	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)

	// Check balance on m[0]
	balance, err := model.LoadCanonicalBalanceByAssetHolder(m[0].Ctx,
		a[0].Name, u[1].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(10), (*big.Int)(&balance.Value))

	// Check balance on m[1]
	balance, err = model.LoadCanonicalBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[2].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(10), (*big.Int)(&balance.Value))
}
