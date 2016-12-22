package app

import (
	"context"
	"fmt"

	goji "goji.io"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/register"
)

// BackgroundContextFromFlags initializes a background context fully loaded
// with everything that could be extracted from the flags.
func BackgroundContextFromFlags(
	envFlag string, // environment
	dsnFlag string, // register DSN
	crdFlag string, // credentials URL
	mntFlag string, // mint host
	mdsFlag string, // mint DSN
	smlFlag string, // SMTP login
	smpFlag string, // SMTP password
	smhFlag string, // SMTP host
	rcpFlag string, // reCAPTCHA secret
) (context.Context, error) {
	ctx := context.Background()

	mintEnv := env.Env{
		Environment: env.QA,
		Config:      map[env.ConfigKey]string{},
	}
	if envFlag == "production" || envFlag == "prod" {
		mintEnv.Environment = env.Production
	}

	mintEnv.Config[register.EnvCfgCredsURL] = crdFlag

	mintEnv.Config[register.EnvCfgMint] = mntFlag

	mintEnv.Config[register.EnvCfgSMTPLogin] = smlFlag
	mintEnv.Config[register.EnvCfgSMTPPassword] = smpFlag
	mintEnv.Config[register.EnvCfgSMTPHost] = smhFlag

	mintEnv.Config[register.EnvCfgReCAPTCHASecret] = rcpFlag

	ctx = env.With(ctx, &mintEnv)

	// registerDB is the DB backing the register service.
	registerDB, err := db.NewDBForDSN(ctx,
		fmt.Sprintf("sqlite3://~/.mint/register-%s.db",
			env.Get(ctx).Environment),
		dsnFlag)
	if err != nil {
		return nil, err
	}
	err = db.CreateDBTables(ctx, "register", registerDB)
	if err != nil {
		return nil, err
	}
	ctx = db.WithDB(ctx, "register", registerDB)

	// mintDB is the DB of the mint this register service is bound to. It is
	// used to create users on the mint once their registration is complete.
	// The tables don't get created here as we want to mimimize the
	// interference with the mintDB.
	mintDB, err := db.NewDBForDSN(ctx,
		fmt.Sprintf("sqlite3://~/.mint/mint-%s.db",
			env.Get(ctx).Environment),
		dsnFlag)
	if err != nil {
		return nil, err
	}
	ctx = db.WithDB(ctx, "mint", mintDB)

	return ctx, nil
}

// Build initializes the app and its web stack.
func Build(
	ctx context.Context,
) (*goji.Mux, error) {
	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDBMap(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))

	logging.Logf(ctx, "Initializing: environment=%s mint=%s",
		env.Get(ctx).Environment, register.GetMint(ctx))

	(&Controller{}).Bind(mux)

	return mux, nil
}
