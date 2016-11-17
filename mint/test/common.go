package test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/app"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"
	goji "goji.io"
)

func init() {
	// Explicitely reproducible
	rand.Seed(1)
}

// Mint represents a test mint.
type Mint struct {
	Server *httptest.Server
	Env    *env.Env
	DB     *sqlx.DB
	Ctx    context.Context
}

// CreateMint creates a new test mint with an in-memory DB and returns
// test.Mint object.
func CreateMint(
	t *testing.T,
) *Mint {
	ctx := context.Background()

	mintEnv := env.Env{
		Environment: env.QA,
		Config:      map[env.ConfigKey]string{},
	}
	ctx = env.With(ctx, &mintEnv)

	mintDB, err := db.NewSqlite3DBInMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = model.CreateMintDBTables(ctx, mintDB)
	if err != nil {
		t.Fatal(err)
	}
	ctx = db.WithDB(ctx, mintDB)

	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDB(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))
	mux.Use(authentication.Middleware)

	(&app.Controller{}).Bind(mux)

	m := Mint{
		Server: httptest.NewServer(mux),
		Env:    &mintEnv,
		DB:     mintDB,
		Ctx:    ctx,
	}
	m.Env.Config[mint.EnvCfgMintHost] = m.Server.URL[7:]

	logging.Logf(ctx, "Creating test mint: minst_host=%s",
		m.Env.Config[mint.EnvCfgMintHost])

	return &m
}

// MintUser reprensents a user of a mint, generally generated by CreateUser.
type MintUser struct {
	Mint     *Mint
	Username string
	Password string
	Address  string
}

var userFirstnames = []string{"kurt", "alan", "albert", "john"}

// CreateUser creates a user and generates an associated MintUser
func (m *Mint) CreateUser(
	t *testing.T,
) *MintUser {
	username := token.New(userFirstnames[rand.Intn(len(userFirstnames))])
	password := token.New("password")

	_, err := model.CreateUser(m.Ctx, username, password)
	if err != nil {
		t.Fatal(err)
	}
	m.Env.Config[mint.EnvCfgMintHost] = m.Server.URL[7:]

	logging.Logf(m.Ctx, "Creating test mint: minst_host=%s",
		m.Env.Config[mint.EnvCfgMintHost])

	return &MintUser{
		m, username, password,
		fmt.Sprintf("%s@%s", username, m.Env.Config[mint.EnvCfgMintHost]),
	}
}

// Post posts to a specified endpoint on the mint.
func (m *Mint) Post(
	t *testing.T,
	user *MintUser,
	path string,
	params url.Values,
) (int, svc.Resp) {
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s%s", m.Server.URL, path),
		strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(user.Username, user.Password)

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}

	return r.StatusCode, raw
}
