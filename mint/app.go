package mint

import (
	"context"

	"goji.io"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/mint/model"

	// force initialization of schemas
	_ "github.com/spolu/settle/mint/model/schemas"
)

// BackgroundContextFromFlags initializes a background context fully loaded
// with everything that could be extracted from the flags.
func BackgroundContextFromFlags(
	envFlag string,
	dbpFlag string,
	hstFlag string,
) (context.Context, error) {
	ctx := context.Background()

	mintEnv := env.Env{
		Environment: env.QA,
		Config:      map[env.ConfigKey]string{},
	}
	if envFlag == "production" {
		mintEnv.Environment = env.Production
	}
	mintEnv.Config[EnvCfgMintHost] = hstFlag
	ctx = env.With(ctx, &mintEnv)

	mintDB, err := db.NewSqlite3DBForPath(ctx, dbpFlag)
	if err != nil {
		return nil, err
	}
	err = model.CreateMintDBTables(ctx, mintDB)
	if err != nil {
		return nil, err
	}
	ctx = db.WithDB(ctx, mintDB)

	return ctx, nil
}

// Build initializes the app and its web stack.
func Build(
	ctx context.Context,
) (*goji.Mux, error) {
	if env.Get(ctx).Config[EnvCfgMintHost] == "" {
		if env.Get(ctx).Environment == env.Production {
			return nil, errors.Newf(
				"You must set the flag `-mint_host` to an externally accessible hostname that other mints can use to contact this mint over HTTPS. If you're just testing and don't have an SSL certificate, please run with `-env=qa`",
			)
		}
		return nil, errors.Newf(
			"You must set the flag `-mint_host` to the hostname that other mints can use to contact this mint over HTTP (since you're running in QA). You can use `-mint_host=127.0.0.1:2407` for testing purposes.",
		)
	}

	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDB(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))
	mux.Use(authentication.Middleware)

	a := &Configuration{}

	logging.Logf(ctx, "Initializing: environment=%s mint_host=%s",
		env.Get(ctx).Environment, env.Get(ctx).Config[EnvCfgMintHost])

	err := a.Init()
	if err != nil {
		return nil, errors.Trace(err)
	}
	a.Bind(mux)

	return mux, nil
}
