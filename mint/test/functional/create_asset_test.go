package functional

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupCreateAsset(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser) {
	m := []*test.Mint{
		test.CreateMint(t),
	}

	u := []*test.MintUser{
		m[0].CreateUser(t),
	}

	return m, u
}

func tearDownCreateAsset(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestCreateAsset(
	t *testing.T,
) {
	t.Parallel()
	m, u := setupCreateAsset(t)
	defer tearDownCreateAsset(t, m)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})

	var asset mint.AssetResource
	err := raw.Extract("asset", &asset)
	assert.Nil(t, err)

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, asset.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, asset.Created*mint.TimeResolutionNs), test.PostLatency)
	assert.Equal(t, u[0].Address, asset.Owner)

	assert.Regexp(t, mint.AssetNameRegexp, asset.Name)
	assert.Equal(t, fmt.Sprintf("%s[USD.2]", u[0].Address), asset.Name)
	assert.Equal(t, "USD", asset.Code)
	assert.Equal(t, int8(2), asset.Scale)
}

func TestCreateAssetWithInvalidCode(
	t *testing.T,
) {
	t.Parallel()
	m, u := setupCreateAsset(t)
	defer tearDownCreateAsset(t, m)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"U/S[D"},
			"scale": {"2"},
		})

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "code_invalid", e.ErrCode)
}

func TestCreateAssetWithInvalidScale(
	t *testing.T,
) {
	t.Parallel()
	m, u := setupCreateAsset(t)
	defer tearDownCreateAsset(t, m)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"221323132122"},
		})

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "scale_invalid", e.ErrCode)
}

func TestCreateAssetThatAlreadyExists(
	t *testing.T,
) {
	t.Parallel()
	m, u := setupCreateAsset(t)
	defer tearDownCreateAsset(t, m)

	status, _ := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})
	assert.Equal(t, 201, status)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})

	var e errors.ConcreteUserError
	err := raw.Extract("error", &e)
	assert.Nil(t, err)

	assert.Equal(t, 400, status)
	assert.Equal(t, "asset_already_exists", e.ErrCode)
}
