package functional

import (
	"fmt"
	"testing"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupRetrieveAsset(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser, []mint.AssetResource) {
	m := []*test.Mint{
		test.CreateMint(t),
	}
	u := []*test.MintUser{
		m[0].CreateUser(t),
	}
	a := []mint.AssetResource{
		u[0].CreateAsset(t, "USD", 2),
		u[0].CreateAsset(t, "EUR", 2),
	}

	return m, u, a
}

func tearDownRetrieveAsset(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestRetrieveAssetSimple(
	t *testing.T,
) {
	t.Parallel()
	m, _, a := setupRetrieveAsset(t)
	defer tearDownRetrieveAsset(t, m)

	status, raw := m[0].Get(t, nil,
		fmt.Sprintf("/assets/%s", a[0].Name))

	var asset mint.AssetResource
	err := raw.Extract("asset", &asset)
	assert.Nil(t, err)

	assert.Equal(t, 200, status)
}

func TestRetrieveAssetDoesNotExist(
	t *testing.T,
) {
	t.Parallel()
	m, _, _ := setupRetrieveAsset(t)
	defer tearDownRetrieveAsset(t, m)

	status, raw := m[0].Get(t, nil,
		fmt.Sprintf("/assets/%s", "albert@princetown.edu[FOO.7]"))

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 404, status)
	assert.Equal(t, "asset_not_found", e.ErrCode)
}
