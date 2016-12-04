package functional

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupPropagateOffer(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser, []mint.AssetResource) {
	m := []*test.Mint{
		test.CreateMint(t),
		test.CreateMint(t),
	}
	u := []*test.MintUser{
		m[0].CreateUser(t),
		m[1].CreateUser(t),
	}
	a := []mint.AssetResource{
		u[0].CreateAsset(t, "USD", 2),
		u[1].CreateAsset(t, "USD", 2),
	}

	return m, u, a
}

func tearDownPropagateOffer(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestPropagateOfferSimple(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupPropagateOffer(t)
	defer tearDownPropagateOffer(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/offers"),
		url.Values{
			"pair":   {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[1].Address)},
			"price":  {"1/1"},
			"amount": {"100"},
		})

	var offer mint.OfferResource
	err := raw.Extract("offer", &offer)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)

	status, raw = m[1].Post(t, nil,
		fmt.Sprintf("/offers/%s", offer.ID),
		url.Values{})

	var prop mint.OfferResource
	err = raw.Extract("offer", &prop)
	assert.Nil(t, err)

	assert.Equal(t, offer.ID, prop.ID)
	assert.Equal(t, offer.Created, prop.Created)
	assert.Equal(t, offer.Owner, prop.Owner)
	assert.Equal(t, offer.Pair, prop.Pair)
	assert.Equal(t, offer.Price, prop.Price)
	assert.Equal(t, offer.Amount, prop.Amount)
	assert.Equal(t, offer.Status, prop.Status)
	assert.Equal(t, offer.Remainder, prop.Remainder)
}
