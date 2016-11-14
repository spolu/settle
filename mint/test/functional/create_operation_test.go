package functional

import (
	"net/url"
	"testing"

	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
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

func TestCreateOperation(
	t *testing.T,
) {
	// m, user, asset := setupWithMintUserAsset(t)

	// status, raw := m.Post(t, user,
	// 	fmt.Sprintf("/assets/%s/operations", asset.Name),
	// 	url.Values{
	// 		"code":  {"USD"},
	// 		"scale": {"2"},
	// 	})

	// var asset mint.AssetResource
	// if err := raw.Extract("asset", &asset); err != nil {
	// 	t.Fatal(err)
	// }
}
