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

func setupCreateOperation(
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
	}

	return m, u, a
}

func tearDownCreateOperation(
	t *testing.T,
	mints []*test.Mint,
) {
	for _, m := range mints {
		m.Close()
	}
}

func TestCreateNoopOperation(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"100"},
			"source":      {u[0].Address},
			"destination": {u[0].Address},
		})

	var op mint.OperationResource
	if err := raw.Extract("operation", &op); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, op.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, op.Created*1000*1000), test.PostLatency)
	assert.Equal(t, u[0].Address, op.Owner)

	assert.Regexp(t, mint.IDRegexp, op.Asset.ID)
	assert.Equal(t, u[0].Address, op.Asset.Owner)
	assert.Regexp(t, mint.AssetNameRegexp, op.Asset.Name)
	assert.Equal(t, fmt.Sprintf("%s[USD.2]", u[0].Address), op.Asset.Name)
	assert.Equal(t, "USD", op.Asset.Code)

	assert.NotNil(t, op.Destination)
	assert.Equal(t, u[0].Address, op.Destination)
	assert.Equal(t, u[0].Address, op.Source)
	assert.Equal(t, int8(2), op.Asset.Scale)
	assert.Equal(t, big.NewInt(100), op.Amount)
}

func TestCreateOperationIssuing(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"10"},
			"source":      {u[0].Address},
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
		time.Unix(0, op.Created*1000*1000), test.PostLatency)
	assert.Equal(t, u[0].Address, op.Owner)

	assert.NotNil(t, op.Source)
	assert.Equal(t, u[0].Address, op.Source)
	assert.NotNil(t, op.Destination)
	assert.Equal(t, "von.neumann@ias.edu", op.Destination)
	assert.Equal(t, int8(2), op.Asset.Scale)
	assert.Equal(t, big.NewInt(10), op.Amount)
}

func TestCreateOperationAnnihilating(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	status, _ := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"10"},
			"source":      {u[0].Address},
			"destination": {"von.neumann@ias.edu"},
		})
	assert.Equal(t, 201, status)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"5"},
			"source":      {"von.neumann@ias.edu"},
			"destination": {u[0].Address},
		})

	var op mint.OperationResource
	if err := raw.Extract("operation", &op); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, op.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, op.Created*1000*1000), test.PostLatency)
	assert.Equal(t, u[0].Address, op.Owner)

	assert.NotNil(t, op.Source)
	assert.Equal(t, "von.neumann@ias.edu", op.Source)
	assert.Equal(t, u[0].Address, op.Destination)
	assert.Equal(t, int8(2), op.Asset.Scale)
	assert.Equal(t, big.NewInt(5), op.Amount)
}

func TestCreateOperationWithNegativeAmount(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"-100"},
			"source":      {u[0].Address},
			"destination": {"von.neumann@ias.edu"},
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
	t.Parallel()
	m, u, _ := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	invalidAsset := "foo"
	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"source":      {u[0].Address},
			"destination": {"von.neumann@ias.edu"},
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
	t.Parallel()
	m, u, _ := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	invalidAsset := "foo@bar.com[USD.2]"
	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"source":      {u[0].Address},
			"destination": {"von.neumann@ias.edu"},
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
	t.Parallel()
	m, u, _ := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	invalidAsset := fmt.Sprintf(
		"foo@%s[USD.2]", m[0].Env.Config[mint.EnvCfgMintHost])
	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"source":      {u[0].Address},
			"destination": {"von.neumann@ias.edu"},
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
	t.Parallel()
	m, u, _ := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	invalidAsset := fmt.Sprintf(
		"%s@%s[FOO.2]", u[0].Username, m[0].Env.Config[mint.EnvCfgMintHost])
	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", invalidAsset),
		url.Values{
			"amount":      {"100"},
			"source":      {u[0].Address},
			"destination": {"von.neumann@ias.edu"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "asset_not_found", e.ErrCode)
}

func TestCreateOperationWithNoSource(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"100"},
			"destination": {"von.neumann@ias.edu"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "source_invalid", e.ErrCode)
}

func TestCreateOperationWithNoDestination(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount": {"100"},
			"source": {u[0].Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "destination_invalid", e.ErrCode)
}

func TestCreateOperationWithInsufficientBalance(
	t *testing.T,
) {
	t.Parallel()
	m, u, a := setupCreateOperation(t)
	defer tearDownCreateOperation(t, m)

	status, _ := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"10"},
			"source":      {u[0].Address},
			"destination": {"von.neumann@ias.edu"},
		})
	assert.Equal(t, 201, status)

	status, raw := u[0].Post(t,
		fmt.Sprintf("/assets/%s/operations", a[0].Name),
		url.Values{
			"amount":      {"100"},
			"source":      {"von.neumann@ias.edu"},
			"destination": {u[0].Address},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "amount_invalid", e.ErrCode)
}
