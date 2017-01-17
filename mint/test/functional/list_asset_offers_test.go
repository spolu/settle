package functional

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupListAssetOffers(
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
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[1].Address),
			"100/100", big.NewInt(100)),
		u[0].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[2].Address),
			"100/100", big.NewInt(100)),
		u[1].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[1].Address, u[0].Address),
			"100/100", big.NewInt(100)),
		u[2].CreateOffer(t,
			fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[2].Address, u[0].Address),
			"98/100", big.NewInt(100)),
	}

	// Propagate m[1], m[2] offers to m[0].
	async.TestRunOne(m[1].Ctx)
	async.TestRunOne(m[2].Ctx)

	return m, u, a, o
}

func tearDownListAssetOffers(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestListAssetOffersSimple(
	t *testing.T,
) {
	t.Parallel()
	m, _, a, o := setupListAssetOffers(t)
	defer tearDownListAssetOffers(t, m)

	status, raw := m[0].Get(t, nil,
		fmt.Sprintf("/assets/%s/offers?propagation=canonical", a[0].Name))

	var offers []mint.OfferResource
	err := raw.Extract("offers", &offers)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(offers))

	assert.Equal(t, o[1].Pair, offers[0].Pair)
	assert.Equal(t, o[1].Price, offers[0].Price)
	assert.Equal(t, o[1].Amount, offers[0].Amount)

	assert.Equal(t, o[0].Pair, offers[1].Pair)
	assert.Equal(t, o[0].Price, offers[1].Price)
	assert.Equal(t, o[0].Amount, offers[1].Amount)

	status, raw = m[0].Get(t, nil,
		fmt.Sprintf("/assets/%s/offers?propagation=propagated", a[0].Name))

	err = raw.Extract("offers", &offers)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(offers))

	assert.Equal(t, o[3].Pair, offers[0].Pair)
	assert.Equal(t, o[3].Price, offers[0].Price)
	assert.Equal(t, o[3].Amount, offers[0].Amount)

	assert.Equal(t, o[2].Pair, offers[1].Pair)
	assert.Equal(t, o[2].Price, offers[1].Price)
	assert.Equal(t, o[2].Amount, offers[1].Amount)

	assert.Equal(t, 200, status)
}

func TestListAssetOffersNoPropagation(
	t *testing.T,
) {
	t.Parallel()
	m, _, a, _ := setupListAssetOffers(t)
	defer tearDownListAssetOffers(t, m)

	status, raw := m[0].Get(t, nil,
		fmt.Sprintf("/assets/%s/offers", a[0].Name))

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "propagation_invalid", e.ErrCode)
}
