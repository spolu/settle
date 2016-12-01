package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
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
	m, u, a, o := setupSettleTransaction(t)
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

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	var tx0 mint.TransactionResource
	if err := raw.Extract("transaction", &tx0); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStSettled, tx0.Status)
	assert.Equal(t, 1, len(tx0.Operations))
	assert.Equal(t, []mint.CrossingResource{}, tx0.Crossings)

	assert.Equal(t, mint.TxStSettled, tx0.Operations[0].Status)

	// Check transaction on m[1].
	status, raw = u[1].Get(t, fmt.Sprintf("/transactions/%s", tx0.ID))

	var tx1 mint.TransactionResource
	if err := raw.Extract("transaction", &tx1); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStSettled, tx1.Status)
	assert.Equal(t, 1, len(tx1.Operations))
	assert.Equal(t, 1, len(tx1.Crossings))

	assert.Equal(t, mint.TxStSettled, tx1.Crossings[0].Status)
	assert.Equal(t, mint.TxStSettled, tx1.Operations[0].Status)

	// Check transaction on m[2].
	status, raw = u[2].Get(t, fmt.Sprintf("/transactions/%s", tx0.ID))

	var tx2 mint.TransactionResource
	if err := raw.Extract("transaction", &tx2); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStSettled, tx2.Status)
	assert.Equal(t, 1, len(tx2.Operations))
	assert.Equal(t, 1, len(tx2.Crossings))

	assert.Equal(t, mint.TxStSettled, tx2.Crossings[0].Status)
	assert.Equal(t, mint.TxStSettled, tx2.Operations[0].Status)

	// Check balance on m[0]
	balance, err := model.LoadBalanceByAssetHolder(m[0].Ctx,
		a[0].Name, u[1].Address)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, big.NewInt(11), (*big.Int)(&balance.Value))

	// Check balance on m[1]
	balance, err = model.LoadBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[2].Address)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, big.NewInt(11), (*big.Int)(&balance.Value))

	// Check that re-settling does not trigger an error.
	status, _ = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	assert.Equal(t, 200, status)
}

func TestSettleTransactionWithWrongSecret(
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

	status, raw = m[1].Post(t, nil,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{
			"hop":    {"1"},
			"secret": {"foo"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "secret_invalid", e.ErrCode)
}
