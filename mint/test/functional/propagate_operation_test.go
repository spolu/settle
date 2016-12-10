package functional

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/model"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupPropagateOperation(
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

func tearDownPropagateOperation(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestPropagateOperationSimple(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupPropagateOffer(t)
	defer tearDownPropagateOffer(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[0].Address)},
			"amount":      {"10"},
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

	async.TestRunOne(m[0].Ctx)
	async.TestRunOne(m[0].Ctx)

	owner, token, err := mint.NormalizedOwnerAndTokenFromID(m[1].Ctx,
		tx.Operations[0].ID)
	assert.Nil(t, err)

	op, err := model.LoadPropagatedOperationByOwnerToken(m[1].Ctx,
		owner, token)
	assert.Nil(t, err)

	assert.Equal(t, owner, op.Owner)
	assert.Equal(t, token, op.Token)
}

func TestPropagateOperationNotSettled(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupPropagateOffer(t)
	defer tearDownPropagateOffer(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/transactions"),
		url.Values{
			"pair":        {fmt.Sprintf("%s[USD.2]/%s[USD.2]", u[0].Address, u[0].Address)},
			"amount":      {"10"},
			"destination": {u[1].Address},
			"path[]":      {},
		})

	var tx mint.TransactionResource
	err := raw.Extract("transaction", &tx)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)

	status, raw = m[1].Post(t, nil,
		fmt.Sprintf("/operations/%s", tx.Operations[0].ID),
		url.Values{})

	var e errors.ConcreteUserError
	err = raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 402, status)
	assert.Equal(t, "propagation_failed", e.ErrCode)
}
