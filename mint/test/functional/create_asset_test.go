package functional

import (
	"context"
	"net/url"
	"testing"

	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
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

	logging.Logf(ctx, "Asset: %q", asset)
}
