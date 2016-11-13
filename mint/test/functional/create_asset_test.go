package functional

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func TestCreateAssetSimple(
	t *testing.T,
) {
	ctx := context.Background()

	m := test.CreateMint(t)
	user := m.CreateUser(t)

	_, raw := m.Post(t, user, "/assets", url.Values{
		"code":  {"USD"},
		"scale": {"2"},
	})

	var asset mint.AssetResource
	if err := raw.Extract("asset", &asset); err != nil {
		t.Fatal(err)
	}

	assert.Regexp(t, mint.IDRegexp, asset.ID)
	assert.WithinDuration(t,
		time.Now(), time.Unix(0, asset.Created*1000*1000), 2*time.Millisecond)
	assert.Equal(t,
		fmt.Sprintf("%s@%s", user.Username, m.Env.Config[mint.EnvCfgMintHost]),
		asset.Issuer)
	assert.Regexp(t, mint.AssetNameRegexp, asset.Name)
	assert.Equal(t,
		fmt.Sprintf("%s@%s[USD.2]",
			user.Username, m.Env.Config[mint.EnvCfgMintHost]),
		asset.Name)
	assert.Equal(t, "USD", asset.Code)
	assert.Equal(t, int8(2), asset.Scale)
}
