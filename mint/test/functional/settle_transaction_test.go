package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupSettleTransaction(
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
			"100/98", big.NewInt(100)),
	}

	return m, u, a, o
}

func tearDownSettleTransaction(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestSettleTransactionWith2Offers(
	t *testing.T,
) {
	t.Parallel()
	m, u, _, o := setupSettleTransaction(t)
	defer tearDownSettleTransaction(t, m)

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

	assert.Equal(t, 201, status)

	var tx mint.TransactionResource
	if err := raw.Extract("transaction", &tx); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("\n>>>>>>>>>>>>>>>>\n\n")

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	var tx0 mint.TransactionResource
	if err := raw.Extract("transaction", &tx0); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, status)
}
