package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupCreateOffer(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser, []mint.AssetResource) {
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

	return m, u, a
}

func tearDownCreateOffer(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestCreateOfferSimple(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupCreateOffer(t)
	defer tearDownCreateOffer(t, m)

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
	assert.Regexp(t, mint.IDRegexp, offer.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, offer.Created*mint.TimeResolutionNs), test.PostLatency)
	assert.Equal(t, u[0].Address, offer.Owner)

	assert.Equal(t,
		fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[1].Address),
		offer.Pair)
	assert.Equal(t, "1/1", offer.Price)
	assert.Equal(t, big.NewInt(100), offer.Amount)
}

func TestCreateOfferWithNotOwnedAsset(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupCreateOffer(t)
	defer tearDownCreateOffer(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/offers"),
		url.Values{
			"pair":   {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[2].Address, u[1].Address)},
			"price":  {"1/1"},
			"amount": {"100"},
		})

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "not_authorized", e.ErrCode)
}

func TestCreateOfferWithInexistantAsset(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupCreateOffer(t)
	defer tearDownCreateOffer(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/offers"),
		url.Values{
			"pair":   {fmt.Sprintf("%s[USD.4]/%s[USD.2]", u[0].Address, u[1].Address)},
			"price":  {"1/1"},
			"amount": {"100"},
		})

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "asset_not_found", e.ErrCode)
}

func TestCreateOfferWithSameAsset(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupCreateOffer(t)
	defer tearDownCreateOffer(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/offers"),
		url.Values{
			"pair":   {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[0].Address)},
			"price":  {"1/1"},
			"amount": {"100"},
		})

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "pair_invalid", e.ErrCode)
}
