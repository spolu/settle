package functional

import (
	"testing"

	"github.com/spolu/settle/mint/test"
)

func TestCreateAssetSimple(
	t *testing.T,
) {
	//ctx := context.Background()

	mint := test.CreateMint(t)
	user := mint.CreateUser(t)

	_ = user

	//mint.Post(ctx, "/assets", map[string]string{
	//	"code":  "USD",
	//	"scale": "2",
	//})
}
