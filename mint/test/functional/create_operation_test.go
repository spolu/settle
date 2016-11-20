package functional

import (
	"fmt"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupWithMintUserAsset(
	t *testing.T,
) (*test.Mint, *test.MintUser, mint.AssetResource) {
	m := test.CreateMint(t)
	user := m.CreateUser(t)

	_, raw := m.Post(t, user,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})

	var asset mint.AssetResource
	if err := raw.Extract("asset", &asset); err != nil {
		t.Fatal(err)
	}

	return m, user, asset
}

func TestCreateIssuingOperation(
	t *testing.T,
) {
	m, user, asset := setupWithMintUserAsset(t)

	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount":      {"100"},
			"destination": {user.Address},
		})

	var op mint.OperationResource
	if err := raw.Extract("operation", &op); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, op.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, op.Created*1000*1000), 2*time.Millisecond)

	assert.Regexp(t, mint.IDRegexp, op.Asset.ID)
	assert.Equal(t, user.Address, op.Asset.Owner)
	assert.Regexp(t, mint.AssetNameRegexp, op.Asset.Name)
	assert.Equal(t, fmt.Sprintf("%s[USD.2]", user.Address), op.Asset.Name)
	assert.Equal(t, "USD", op.Asset.Code)

	assert.NotNil(t, op.Destination)
	assert.Equal(t, user.Address, *op.Destination)
	assert.Nil(t, op.Source)
	assert.Equal(t, int8(2), op.Asset.Scale)
	assert.Equal(t, big.NewInt(100), op.Amount)
}

func TestCreateOperation(
	t *testing.T,
) {
	m, user, asset := setupWithMintUserAsset(t)

	status, _ := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount":      {"100"},
			"destination": {user.Address},
		})

	assert.Equal(t, 201, status)

	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount":      {"10"},
			"source":      {user.Address},
			"destination": {"von.neumann@ias.edu"},
		})

	var op mint.OperationResource
	if err := raw.Extract("operation", &op); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, op.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, op.Created*1000*1000), 2*time.Millisecond)

	assert.NotNil(t, op.Source)
	assert.Equal(t, user.Address, *op.Source)
	assert.NotNil(t, op.Destination)
	assert.Equal(t, "von.neumann@ias.edu", *op.Destination)
	assert.Equal(t, int8(2), op.Asset.Scale)
	assert.Equal(t, big.NewInt(10), op.Amount)
}

func TestCreateAnnihilatingOperation(
	t *testing.T,
) {
	m, user, asset := setupWithMintUserAsset(t)

	status, _ := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount":      {"100"},
			"destination": {user.Address},
		})

	assert.Equal(t, 201, status)

	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount": {"10"},
			"source": {user.Address},
		})

	var op mint.OperationResource
	if err := raw.Extract("operation", &op); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, op.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, op.Created*1000*1000), 2*time.Millisecond)

	assert.NotNil(t, op.Source)
	assert.Equal(t, user.Address, *op.Source)
	assert.Nil(t, op.Destination)
	assert.Equal(t, int8(2), op.Asset.Scale)
	assert.Equal(t, big.NewInt(10), op.Amount)
}

func TestCreateOperationWithNegativeAmount(
	t *testing.T,
) {
	m, user, asset := setupWithMintUserAsset(t)

	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount":      {"-100"},
			"destination": {user.Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "amount_invalid", e.ErrCode)
}

func TestCreateOperationWithInvalidAsset(
	t *testing.T,
) {
	m, user, _ := setupWithMintUserAsset(t)

	invalidAsset := "foo"
	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"destination": {user.Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "asset_invalid", e.ErrCode)
}

func TestCreateOperationWithInvalidAssetHostname(
	t *testing.T,
) {
	m, user, _ := setupWithMintUserAsset(t)

	invalidAsset := "foo@bar.com[USD.2]"
	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"destination": {user.Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "operation_not_authorized", e.ErrCode)
}

func TestCreateOperationWithInvalidAssetUsername(
	t *testing.T,
) {
	m, user, _ := setupWithMintUserAsset(t)

	invalidAsset := fmt.Sprintf(
		"foo@%s[USD.2]", m.Env.Config[mint.EnvCfgMintHost])
	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"destination": {user.Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "operation_not_authorized", e.ErrCode)
}

func TestCreateOperationWithUnknownAsset(
	t *testing.T,
) {
	m, user, _ := setupWithMintUserAsset(t)

	invalidAsset := fmt.Sprintf(
		"%s@%s[FOO.2]", user.Username, m.Env.Config[mint.EnvCfgMintHost])
	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"destination": {user.Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "asset_not_found", e.ErrCode)
}

func TestCreateOperationWithNoSourceOrDestination(
	t *testing.T,
) {
	m, user, asset := setupWithMintUserAsset(t)

	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount": {"100"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "operation_invalid", e.ErrCode)
}

func TestCreateOperationWithInsufficientBalance(
	t *testing.T,
) {
	m, user, asset := setupWithMintUserAsset(t)

	status, _ := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount":      {"10"},
			"destination": {"von.neumann@ias.edu"},
		})
	assert.Equal(t, 201, status)

	status, raw := m.Post(t, user,
		fmt.Sprintf("/assets/%s/operations", asset.Name),
		url.Values{
			"amount":      {"100"},
			"source":      {"von.neumann@ias.edu"},
			"destination": {user.Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "amount_invalid", e.ErrCode)
}
