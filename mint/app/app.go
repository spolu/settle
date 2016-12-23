package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"goji.io"

	"github.com/spolu/settle/lib/cert"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/lib/authentication"
	"github.com/spolu/settle/register"

	// force initialization of schemas
	_ "github.com/spolu/settle/mint/model/schemas"
)

// BackgroundContextFromFlags initializes a background context fully loaded
// with everything that could be extracted from the flags.
func BackgroundContextFromFlags(
	envFlag string,
	dsnFlag string,
	hstFlag string,
	prtFlag string,
	keyFlag string,
	crtFlag string,
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

	port := fmt.Sprintf("%d", mint.DefaultPort[mintEnv.Environment])
	if prtFlag != "" {
		port = prtFlag
	}
	mintEnv.Config[mint.EnvCfgPort] = port
	mintEnv.Config[register.EnvCfgKeyFile] = keyFlag
	mintEnv.Config[register.EnvCfgCrtFile] = crtFlag

	ctx = env.With(ctx, &mintEnv)

	mintDB, err := db.NewDBForDSN(ctx,
		fmt.Sprintf("sqlite3://~/.mint/mint-%s.db",
			env.Get(ctx).Environment),
		dsnFlag)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = db.CreateDBTables(ctx, "mint", mintDB)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ctx = db.WithDB(ctx, "mint", mintDB)

	a, err := async.NewAsync(ctx)
	if err != nil {
		return nil, errors.Trace(err)
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
			return nil, errors.Trace(errors.Newf(
				"You must set the `-host` flag to an publicly accessible hostname that other mints can use to contact this mint over HTTPS (SSL certificates will be automatically generated from `Let's Encrypt` in production). If you're just testing and don't have a public domain name pointing to this machine, please run with `-env=qa` and `-host=127.0.0.1`",
			))
		}
		return nil, errors.Trace(errors.Newf(
			"You must set the `-host` flag to the hostname that other mints can use to contact this mint over HTTP (since you're running in QA). You can use `-host=127.0.0.1` for testing purposes.",
		))
	}

	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(db.Middleware(db.GetDBMap(ctx)))
	mux.Use(env.Middleware(env.Get(ctx)))
	mux.Use(async.Middleware(async.Get(ctx)))
	mux.Use(authentication.Middleware)

	logging.Logf(ctx, "Initializing: environment=%s host=%s port=%s",
		env.Get(ctx).Environment, mint.GetHost(ctx), mint.GetPort(ctx))

	(&Controller{}).Bind(mux)

	// Start on async worker.
	go func() {
		async.Get(ctx).Run()
	}()

	return mux, nil
}

// Serve the goji mux.
func Serve(
	ctx context.Context,
	mux *goji.Mux,
) error {

	s := &http.Server{
		Addr:         fmt.Sprintf(":%s", mint.GetPort(ctx)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		TLSConfig: &tls.Config{
			GetCertificate: cert.GetGetCertificate(ctx,
				mint.GetHost(ctx),
				mint.GetCrtFile(ctx), mint.GetKeyFile(ctx)),
			PreferServerCipherSuites: true,
			// Only use curves which have assembly implementations
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				// tls.X25519, // Go 1.8 only
			},
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				// tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				// tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
		Handler: mux,
	}

	logging.Logf(ctx, "Listening: port=%s", mint.GetPort(ctx))

	err := s.ListenAndServeTLS("", "")
	if err != nil {
		return errors.Trace(err)
	}

	// Install our handler at the root of the standard net/http default mux.
	// This allows packages like expvar to continue working as expected.
	// http.Handle("/", mux)

	return nil
}
