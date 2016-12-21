package app

import (
	"context"
	"fmt"

	"goji.io"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/lib/authentication"

	// force initialization of schemas
	_ "github.com/spolu/settle/mint/model/schemas"
)

// BackgroundContextFromFlags initializes a background context fully loaded
// with everything that could be extracted from the flags.
func BackgroundContextFromFlags(
	envFlag string,
	dsnFlag string,
	hstFlag string,
) (context.Context, error) {
	ctx := context.Background()

	mintEnv := env.Env{
		Environment: env.QA,
		Config:      map[env.ConfigKey]string{},
	}
	if envFlag == "production" || envFlag == "prod" {
		mintEnv.Environment = env.Production
	}
	mintEnv.Config[mint.EnvCfgHost] = hstFlag
	ctx = env.With(ctx, &mintEnv)

	mintDB, err := db.NewDBForDSN(ctx,
		fmt.Sprintf("sqlite3://~/.mint/mint-%s.db",
			env.Get(ctx).Environment),
		dsnFlag)
	if err != nil {
		return nil, err
	}
	err = db.CreateDBTables(ctx, "mint", mintDB)
	if err != nil {
		return nil, err
	}
	ctx = db.WithDB(ctx, "mint", mintDB)

	a, err := async.NewAsync(ctx)
	if err != nil {
		return nil, err
	}
	ctx = async.With(ctx, a)

	return ctx, nil
}

// Build initializes the app and its web stack.
func Build(
	ctx context.Context,
) (*goji.Mux, error) {
	if mint.GetHost(ctx) == "" {
		if env.Get(ctx).Environment == env.Production {
			return nil, errors.Newf(
				"You must set the `-host` flag to an publicly accessible hostname that other mints can use to contact this mint over HTTPS (SSL certificates will be automatically generated from `Let's Encrypt` in production). If you're just testing and don't have a public domain name pointing to this machine, please run with `-env=qa` and `-host=127.0.0.1`",
			)
		}
		return nil, errors.Newf(
			"You must set the `-host` flag to the hostname that other mints can use to contact this mint over HTTP (since you're running in QA). You can use `-host=127.0.0.1` for testing purposes.",
		)
	}

	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDBMap(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))
	mux.Use(async.Middleware(async.Get(ctx)))
	mux.Use(authentication.Middleware)

	logging.Logf(ctx, "Initializing: environment=%s mint_host=%s",
		env.Get(ctx).Environment, mint.GetHost(ctx))

	(&Controller{}).Bind(mux)

	// Start on async worker.
	go func() {
		async.Get(ctx).Run()
	}()

	return mux, nil
}
