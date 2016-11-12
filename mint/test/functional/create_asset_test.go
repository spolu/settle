package functional

import (
	"testing"

	"github.com/spolu/settle/mint/test"
)

func TestCreateAssetSimple(
	t *testing.T,
) {

	mint, err := test.CreateMint(t)
	if err != nil {
		t.Errorf("create test mint failed: %v", err)
	}
	_ = mint
}
