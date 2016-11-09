package mint

import (
	"context"
	"os"

	"goji.io"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/mint/lib/authentication"

	// force initialization of schemas
	_ "github.com/spolu/settle/mint/model/schemas"
)

// Build initializes the app and its web stack.
func Build() (*goji.Mux, error) {
	mux := goji.NewMux()
	mux.Use(requestlogger.Middleware)
	mux.Use(recoverer.Middleware)
	mux.Use(livemode.Middleware)
	mux.Use(authentication.Middleware)

	err := error(nil)

	if os.Getenv("MINT_HOST") == "" {
		if os.Getenv("ENVIRONMENT") == "production" {
			return nil, errors.Newf(
				"In production, you must set the environment variable " +
					"`MINT_HOST` to the host name under which you want to " +
					"run this mint, for which you must have an SSL " +
					"certificate.",
			)
		}
		return nil, errors.Newf(
			"You must set the environment variable `MINT_HOST` to the host " +
				"name under which you want to run this mint. In QA you don't " +
				"need to provide an SSL certificate for this hostname and " +
				"you can use `localhost:2046` for testing purposes.",
		)
	}

	a := &Configuration{
		MintHost: os.Getenv("MINT_HOST"),
	}

	ctx := context.Background()

	logging.Logf(ctx, "Initializing: environment=%s mint_host=%q",
		env.Current, a.MintHost)

	err = a.Init()
	if err != nil {
		return nil, errors.Trace(err)
	}
	a.Bind(mux)

	return mux, nil
}
