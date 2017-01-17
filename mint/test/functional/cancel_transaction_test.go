package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupCancelTransaction(
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
			fmt.Sprintf("%s/%s", a[0].Name, a[2].Name),
			"100/100", big.NewInt(100)),
		u[1].CreateOffer(t,
			fmt.Sprintf("%s/%s", a[1].Name, a[0].Name),
			"100/100", big.NewInt(100)),
		u[2].CreateOffer(t,
			fmt.Sprintf("%s/%s", a[2].Name, a[1].Name),
			"98/100", big.NewInt(100)),
	}

	return m, u, a, o
}

func tearDownCancelTransaction(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestCancelTransactionWith2Offers(
	t *testing.T,
) {
	t.Parallel()
	m, u, a, o := setupCancelTransaction(t)
	defer tearDownCancelTransaction(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s/%s", a[0].Name, a[2].Name)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]": {
				o[1].ID,
				o[2].ID,
			},
		})

	assert.Equal(t, 201, status)

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	status, raw = u[2].Post(t,
		fmt.Sprintf("/transactions/%s/cancel", tx.ID),
		url.Values{})

	var tx2 mint.TransactionResource
	err = raw.Extract("transaction", &tx2)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStCanceled, tx2.Status)
	assert.Equal(t, 1, len(tx2.Operations))
	assert.Equal(t, 1, len(tx2.Crossings))

	assert.Equal(t, mint.TxStCanceled, tx2.Crossings[0].Status)
	assert.Equal(t, mint.TxStCanceled, tx2.Operations[0].Status)

	// Check transaction on m[1].
	status, raw = u[1].Get(t, fmt.Sprintf("/transactions/%s", tx.ID))

	var tx1 mint.TransactionResource
	err = raw.Extract("transaction", &tx1)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStCanceled, tx1.Status)
	assert.Equal(t, 1, len(tx1.Operations))
	assert.Equal(t, 1, len(tx1.Crossings))

	assert.Equal(t, mint.TxStCanceled, tx1.Crossings[0].Status)
	assert.Equal(t, mint.TxStCanceled, tx1.Operations[0].Status)

	// Check transaction on m[0].
	status, raw = u[0].Get(t, fmt.Sprintf("/transactions/%s", tx.ID))

	var tx0 mint.TransactionResource
	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStCanceled, tx0.Status)
	assert.Equal(t, 0, len(tx0.Crossings))
	assert.Equal(t, 1, len(tx0.Operations))

	assert.Equal(t, mint.TxStCanceled, tx0.Operations[0].Status)

	// Check balance on m[0]
	balance, err := model.LoadCanonicalBalanceByAssetHolder(m[0].Ctx,
		a[0].Name, u[1].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(0), (*big.Int)(&balance.Value))

	// Check balance on m[1]
	balance, err = model.LoadCanonicalBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[2].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(0), (*big.Int)(&balance.Value))

	// Check that re-canceling does not trigger an error.
	status, _ = u[2].Post(t,
		fmt.Sprintf("/transactions/%s/cancel", tx.ID),
		url.Values{})

	assert.Equal(t, 200, status)

	// Check that settling does  trigger an error.
	status, _ = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	assert.Equal(t, 402, status)
}

func TestCancelTransactionmWithNoOffer(
	t *testing.T,
) {
	t.Parallel()
	m, u, a, _ := setupCancelTransaction(t)
	defer tearDownCancelTransaction(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s/%s", a[0].Name, a[0].Name)},
			"amount":      {"10"},
			"destination": {u[1].Address},
		})

	assert.Equal(t, 201, status)

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/cancel", tx.ID),
		url.Values{})

	var tx0 mint.TransactionResource
	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStCanceled, tx0.Status)
	assert.Equal(t, 0, len(tx0.Crossings))
	assert.Equal(t, 1, len(tx0.Operations))

	assert.Equal(t, u[0].Address, tx0.Operations[0].Source)
	assert.Equal(t, u[1].Address, tx0.Operations[0].Destination)
	assert.Equal(t, big.NewInt(10), tx0.Operations[0].Amount)
	assert.Equal(t, mint.TxStCanceled, tx0.Operations[0].Status)
	assert.Equal(t, tx.ID, *tx0.Operations[0].Transaction)
	assert.Equal(t, int8(0), *tx0.Operations[0].TransactionHop)

	// Check balance on m[0]
	balance, err := model.LoadCanonicalBalanceByAssetHolder(m[0].Ctx,
		a[0].Name, u[1].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(0), (*big.Int)(&balance.Value))
}

func TestCancelTransactionWithRemoteBaseAsset(
	t *testing.T,
) {
	t.Parallel()
	m, u, a, o := setupCreateTransaction(t)
	defer tearDownCreateTransaction(t, m)

	// Credit u[0] of u[1] USD.2
	status, raw := u[1].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s/%s", a[1].Name, a[1].Name)},
			"amount":      {"11"},
			"destination": {u[0].Address},
			"path[]":      {},
		})

	assert.Equal(t, 201, status)

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	status, _ = u[1].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})
	assert.Equal(t, 200, status)

	// Attempt to create
	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s/%s", a[1].Name, a[2].Name)},
			"amount":      {"10"},
			"destination": {u[2].Address},
			"path[]": {
				o[2].ID,
			},
		})

	var tx0 mint.TransactionResource
	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	// Check that cancelation can't happen on m[0] and m[1].
	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions/%s/cancel", tx0.ID),
		url.Values{})
	assert.Equal(t, 402, status)
	status, raw = u[1].Post(t,
		fmt.Sprintf("/transactions/%s/cancel", tx0.ID),
		url.Values{})
	assert.Equal(t, 402, status)

	status, raw = u[2].Post(t,
		fmt.Sprintf("/transactions/%s/cancel", tx0.ID),
		url.Values{})

	assert.Equal(t, 200, status)

	var tx2 mint.TransactionResource
	err = raw.Extract("transaction", &tx2)
	assert.Nil(t, err)

	// Check transaction from m[2].
	assert.Equal(t, mint.TxStCanceled, tx2.Status)
	assert.Equal(t, 1, len(tx2.Crossings))
	assert.Equal(t, 1, len(tx2.Operations))

	assert.Equal(t, mint.TxStCanceled, tx2.Crossings[0].Status)
	assert.Equal(t, mint.TxStCanceled, tx2.Operations[0].Status)

	// Check transaction on m[1].
	status, raw = m[1].Get(t, nil, fmt.Sprintf("/transactions/%s", tx0.ID))

	var tx1 mint.TransactionResource
	err = raw.Extract("transaction", &tx1)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)
	assert.Equal(t, mint.TxStCanceled, tx1.Status)
	assert.Equal(t, 0, len(tx1.Crossings))
	assert.Equal(t, 1, len(tx1.Operations))

	assert.Equal(t, mint.TxStCanceled, tx1.Operations[0].Status)

	// Check transaction on m[0].
	status, raw = m[0].Get(t, nil, fmt.Sprintf("/transactions/%s", tx0.ID))

	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	assert.Regexp(t, mint.IDRegexp, tx0.ID)
	assert.Equal(t, mint.TxStCanceled, tx0.Status)
	assert.Equal(t, 0, len(tx0.Operations))
	assert.Equal(t, 0, len(tx0.Crossings))

	// Check balance on m[1]
	balance, err := model.LoadCanonicalBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[0].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(11), (*big.Int)(&balance.Value))

	balance, err = model.LoadCanonicalBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[2].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(0), (*big.Int)(&balance.Value))
}

func TestCancelTransactionWithNoOfferAndRemoteBaseAsset(
	t *testing.T,
) {
	t.Parallel()
	m, u, a, _ := setupCreateTransaction(t)
	defer tearDownCreateTransaction(t, m)

	// Credit u[0] on m[1]
	status, raw := u[1].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s/%s", a[1].Name, a[1].Name)},
			"amount":      {"10"},
			"destination": {u[0].Address},
			"path[]":      {},
		})

	assert.Equal(t, 201, status)

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	status, raw = u[1].Post(t,
		fmt.Sprintf("/transactions/%s/settle", tx.ID),
		url.Values{})

	assert.Equal(t, 200, status)

	err = raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	// Credit u[2] on m[1] from u[0]
	status, raw = u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s/%s", a[1].Name, a[1].Name)},
			"amount":      {"5"},
			"destination": {u[2].Address},
			"path[]":      {},
		})

	assert.Equal(t, 201, status)

	var tx0 mint.TransactionResource
	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	status, raw = u[1].Post(t,
		fmt.Sprintf("/transactions/%s/cancel", tx0.ID),
		url.Values{})

	var tx1 mint.TransactionResource
	err = raw.Extract("transaction", &tx1)
	assert.Nil(t, err)

	assert.Regexp(t, mint.IDRegexp, tx1.ID)
	assert.Equal(t, mint.TxStCanceled, tx1.Status)
	assert.Equal(t, 1, len(tx1.Operations))
	assert.Equal(t, 0, len(tx1.Crossings))

	assert.Equal(t, mint.TxStCanceled, tx1.Operations[0].Status)
	assert.Equal(t, big.NewInt(5), tx1.Operations[0].Amount)
	assert.Equal(t, u[2].Address, tx1.Operations[0].Destination)
	assert.Equal(t, u[0].Address, tx1.Operations[0].Source)

	// Check transaction on m[0].
	status, raw = u[0].Get(t, fmt.Sprintf("/transactions/%s", tx1.ID))

	assert.Equal(t, 200, status)

	err = raw.Extract("transaction", &tx0)
	assert.Nil(t, err)

	assert.Regexp(t, mint.IDRegexp, tx0.ID)
	assert.Equal(t, mint.TxStCanceled, tx0.Status)
	assert.Equal(t, 0, len(tx0.Operations))
	assert.Equal(t, 0, len(tx0.Crossings))

	// Check balance on m[1]
	balance, err := model.LoadCanonicalBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[0].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(10), (*big.Int)(&balance.Value))

	balance, err = model.LoadCanonicalBalanceByAssetHolder(m[1].Ctx,
		a[1].Name, u[2].Address)
	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(0), (*big.Int)(&balance.Value))
}
