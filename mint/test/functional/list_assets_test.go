package functional

import (
	"fmt"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupListAssets(
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
		u[0].CreateAsset(t, "GBP", 2),
		u[0].CreateAsset(t, "KRN", 2),
		u[0].CreateAsset(t, "NGN", 2),
		u[0].CreateAsset(t, "AU-LAIT", 0),
		u[0].CreateAsset(t, "HOUR-OF-WORK", 0),
	}

	return m, u, a
}

func tearDownListAssets(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestListAssetsSimple(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupListAssets(t)
	defer tearDownListAssets(t, m)

	status, raw := u[0].Get(t, fmt.Sprintf("/assets"))

	var assets []mint.AssetResource
	err := raw.Extract("assets", &assets)
	assert.Nil(t, err)

	assert.Equal(t, len(a), len(assets))

	assert.Equal(t, "HOUR-OF-WORK", assets[0].Code)
	assert.Equal(t, int8(0), assets[0].Scale)

	assert.Equal(t, "USD", assets[len(assets)-1].Code)
	assert.Equal(t, int8(2), assets[len(assets)-1].Scale)

	assert.Equal(t, 200, status)
}

func TestListAssetsWithLimit(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupListAssets(t)
	defer tearDownListAssets(t, m)

	status, raw := u[0].Get(t, fmt.Sprintf("/assets?limit=2"))

	var assets []mint.AssetResource
	err := raw.Extract("assets", &assets)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(assets))

	assert.Equal(t, "HOUR-OF-WORK", assets[0].Code)
	assert.Equal(t, int8(0), assets[0].Scale)

	assert.Equal(t, "AU-LAIT", assets[len(assets)-1].Code)
	assert.Equal(t, int8(0), assets[len(assets)-1].Scale)

	assert.Equal(t, 200, status)
}

func TestListAssetsWithLimitAndCreatedBefore(
	t *testing.T,
) {
	t.Parallel()
	m, u, _ := setupListAssets(t)
	defer tearDownListAssets(t, m)

	status, raw := u[0].Get(t, fmt.Sprintf("/assets?limit=2"))

	var assets []mint.AssetResource
	err := raw.Extract("assets", &assets)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(assets))

	assert.Equal(t, "HOUR-OF-WORK", assets[0].Code)
	assert.Equal(t, int8(0), assets[0].Scale)

	assert.Equal(t, "AU-LAIT", assets[1].Code)
	assert.Equal(t, int8(0), assets[1].Scale)

	assert.Equal(t, 200, status)

	status, raw = u[0].Get(t,
		fmt.Sprintf("/assets?limit=2&created_before=%d", assets[1].Created))

	err = raw.Extract("assets", &assets)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(assets))

	assert.Equal(t, "NGN", assets[0].Code)
	assert.Equal(t, int8(2), assets[0].Scale)

	assert.Equal(t, "KRN", assets[1].Code)
	assert.Equal(t, int8(2), assets[1].Scale)
}
