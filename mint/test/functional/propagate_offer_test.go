package functional

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/model"
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

	async.TestRunOne(m[0].Ctx)

	owner, token, err := mint.NormalizedOwnerAndTokenFromID(m[1].Ctx, offer.ID)
	assert.Nil(t, err)

	of, err := model.LoadPropagatedOfferByOwnerToken(m[1].Ctx, owner, token)
	assert.Nil(t, err)

	assert.Equal(t, owner, of.Owner)
	assert.Equal(t, token, of.Token)
}
