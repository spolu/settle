package test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/authentication"
	goji "goji.io"
)

func init() {
	env.Current = env.QA
}

type TestMint struct {
	Server *httptest.Server
}

func CreateTestMint(
	t *testing.T,
) *TestMint {
	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(livemode.Middleware)
	mux.Use(authentication.Middleware)

	port := 2300
	a := &mint.Configuration{
		MintHost: fmt.Sprintf("127.0.0.1:%s", port),
	}
	err := a.Init()
	if err != nil {
		t.Fatalf("test mint setup failed: %+v", err)
	}
	a.Bind(mux)

	return &TestMint{
		Server: httptest.NewServer(mux),
	}
}

func TestCreateAssetSimple(
	t *testing.T,
) {
	mint := CreateTestMint(t)
	_ = mint
}
