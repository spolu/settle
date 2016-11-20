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

func setupWithMintUser(
	t *testing.T,
) (*test.Mint, *test.MintUser) {
	m := test.CreateMint(t)
	user := m.CreateUser(t)

	return m, user
}

func TestCreateAsset(
	t *testing.T,
) {
	m, user := setupWithMintUser(t)

	status, raw := m.Post(t, user,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})

	var asset mint.AssetResource
	if err := raw.Extract("asset", &asset); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, asset.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, asset.Created*1000*1000), 2*time.Millisecond)

	assert.Equal(t, user.Address, asset.Owner)
	assert.Regexp(t, mint.AssetNameRegexp, asset.Name)
	assert.Equal(t, fmt.Sprintf("%s[USD.2]", user.Address), asset.Name)
	assert.Equal(t, "USD", asset.Code)
	assert.Equal(t, int8(2), asset.Scale)
}

func TestCreateAssetWithInvalidCode(
	t *testing.T,
) {
	m, user := setupWithMintUser(t)

	status, raw := m.Post(t, user,
		"/assets",
		url.Values{
			"code":  {"U/S[D"},
			"scale": {"2"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "code_invalid", e.ErrCode)
}

func TestCreateAssetWithInvalidScale(
	t *testing.T,
) {
	m, user := setupWithMintUser(t)

	status, raw := m.Post(t, user,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"221323132122"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "scale_invalid", e.ErrCode)
}

func TestCreateAssetThatAlreadyExists(
	t *testing.T,
) {
	m, user := setupWithMintUser(t)

	status, _ := m.Post(t, user,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})
	assert.Equal(t, 201, status)

	status, raw := m.Post(t, user,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "asset_already_exists", e.ErrCode)
}
