package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupListAssetBalances(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser, []mint.AssetResource) {
	m := []*test.Mint{
		test.CreateMint(t),
		test.CreateMint(t),
	}
	u := []*test.MintUser{
		m[0].CreateUser(t),
		m[1].CreateUser(t),
		m[1].CreateUser(t),
		m[0].CreateUser(t),
	}
	a := []mint.AssetResource{
		u[0].CreateAsset(t, "USD", 2),
	}

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[0].Address)},
			"amount":      {"42"},
			"destination": {u[1].Address},
			"path[]":      {},
		})

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	assert.Equal(t, 200, status)

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[0].Address)},
			"amount":      {"27"},
			"destination": {u[2].Address},
			"path[]":      {},
		})

	err = raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	assert.Equal(t, 200, status)

	return m, u, a
}

func tearDownListAssetBalances(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestListAssetBalancesSimple(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupListAssetBalances(t)
	defer tearDownListAssetBalances(t, m)

	status, raw := u[0].Get(t, fmt.Sprintf("/assets/%s/balances", a[0].Name))

	var balances []mint.BalanceResource
	err := raw.Extract("balances", &balances)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(balances))

	assert.Equal(t, a[0].Name, balances[0].Asset)
	assert.Equal(t, u[2].Address, balances[0].Holder)
	assert.Equal(t, big.NewInt(27), balances[0].Value)

	assert.Equal(t, a[0].Name, balances[1].Asset)
	assert.Equal(t, u[1].Address, balances[1].Holder)
	assert.Equal(t, big.NewInt(42), balances[1].Value)

	assert.Equal(t, 200, status)
}

func TestListAssetBalancesUnauthorized(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupListAssetBalances(t)
	defer tearDownListAssetBalances(t, m)

	status, raw := u[3].Get(t, fmt.Sprintf("/assets/%s/balances", a[0].Name))

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "not_authorized", e.ErrCode)
}
