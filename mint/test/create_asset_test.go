package test

import "testing"

func TestCreateAssetSimple(
	t *testing.T,
) {
	mint, err := CreateTestMint(t)
	if err != nil {
		t.Errorf("create test mint failed: %v", err)
	}
	_ = mint
}
