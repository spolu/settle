package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupListBalances(
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
	}
	a := []mint.AssetResource{
		u[0].CreateAsset(t, "USD", 2),
		u[2].CreateAsset(t, "EUR", 2),
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

	status, raw = u[2].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[EUR.2]/%s[EUR.2]", u[2].Address, u[2].Address)},
			"amount":      {"27"},
			"destination": {u[1].Address},
			"path[]":      {},
		})

	assert.Equal(t, 201, status)

	async.TestRunOne(m[0].Ctx)
	async.TestRunOne(m[0].Ctx)

	return m, u, a
}

func tearDownListBalances(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestListBalancesSimple(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupListBalances(t)
	defer tearDownListBalances(t, m)

	status, raw := u[1].Get(t, fmt.Sprintf("/balances"))

	var balances []mint.BalanceResource
	err := raw.Extract("balances", &balances)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(balances))

	assert.Equal(t, a[1].Name, balances[0].Asset)
	assert.Equal(t, big.NewInt(0), balances[0].Value)

	// Transaction not yet settled, so balance should exist but at 0.
	assert.Equal(t, a[0].Name, balances[1].Asset)
	assert.Equal(t, big.NewInt(42), balances[1].Value)

	assert.Equal(t, 200, status)
}
