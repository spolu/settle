package test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
	goji "goji.io"
)

// Mint represents a test mint.
type Mint struct {
	Server *httptest.Server
}

// CreateTestMint creates a new test mint with an in-memory DB and returns
// test.Mint object.
func CreateTestMint(
	t *testing.T,
) (*Mint, error) {
	ctx := context.Background()

	mintEnv := env.Env{
		Environment: env.QA,
		Config:      map[env.ConfigKey]string{},
	}
	ctx = env.With(ctx, &mintEnv)

	mintDB, err := db.NewSqlite3DBInMemory(ctx)
	if err != nil {
		return nil, err
	}
	err = model.CreateMintDBTables(ctx, mintDB)
	if err != nil {
		return nil, err
	}
	ctx = db.WithDB(ctx, mintDB)

	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDB(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))
	mux.Use(livemode.Middleware)
	mux.Use(authentication.Middleware)

	a := &mint.Configuration{}

	err = a.Init()
	if err != nil {
		return nil, err
	}
	a.Bind(mux)

	m := Mint{
		Server: httptest.NewServer(mux),
	}

	return &m, nil
}
