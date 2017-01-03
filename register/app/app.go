package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	goji "goji.io"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/register"

	// force initialization of schemas
	_ "github.com/spolu/settle/register/model/schemas"
)

// BackgroundContextFromFlags initializes a background context fully loaded
// with everything that could be extracted from the flags.
func BackgroundContextFromFlags(
	envFlag string, // environment
	hstFlag string, // register host
	prtFlag string, // register port
	dsnFlag string, // register DSN
	crdFlag string, // credentials URL
	mntFlag string, // mint host
	mdsFlag string, // mint DSN
	smlFlag string, // SMTP login
	smpFlag string, // SMTP password
	smhFlag string, // SMTP host
	frmFlag string, // from address
) (context.Context, error) {
	ctx := context.Background()

	registerEnv := env.Env{
		Environment: env.QA,
		Config:      map[env.ConfigKey]string{},
	}
	if envFlag == "production" || envFlag == "prod" {
		registerEnv.Environment = env.Production
	}

	registerEnv.Config[register.EnvCfgHost] = hstFlag
	registerEnv.Config[register.EnvCfgPort] = prtFlag

	registerEnv.Config[register.EnvCfgCredsURL] = crdFlag
	registerEnv.Config[register.EnvCfgMint] = mntFlag

	registerEnv.Config[register.EnvCfgSMTPLogin] = smlFlag
	registerEnv.Config[register.EnvCfgSMTPPassword] = smpFlag
	registerEnv.Config[register.EnvCfgSMTPHost] = smhFlag
	registerEnv.Config[register.EnvCfgFrom] = frmFlag

	ctx = env.With(ctx, &registerEnv)

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
	if register.GetHost(ctx) == "" {
		return nil, errors.Trace(errors.Newf(
			"You must set the `-host` flag"))
	}
	if register.GetPort(ctx) == "" {
		return nil, errors.Trace(errors.Newf(
			"You must set the `-port` flag"))
	}
	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDBMap(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))

	logging.Logf(ctx, "Initializing: environment=%s host=%s port=%s mint=%s",
		env.Get(ctx).Environment,
		register.GetHost(ctx), register.GetPort(ctx),
		register.GetMint(ctx))

	(&Controller{}).Bind(mux)

	return mux, nil
}

// Serve the goji mux.
func Serve(
	ctx context.Context,
	mux *goji.Mux,
) error {

	s := &http.Server{
		Addr:         fmt.Sprintf(":%s", register.GetPort(ctx)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler:      mux,
	}

	logging.Logf(ctx, "Listening: port=%s", register.GetPort(ctx))

	err := gracehttp.Serve(s)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
