package mint

import (
	"os"

	"goji.io"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
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
		return nil, errors.Newf(
			"You must set the environment variable `MINT_HOST` to the " +
				"host name under which you want to run this mint.",
		)
	}

	a := &Configuration{
		MintHost: os.Getenv("MINT_HOST"),
	}

	err = a.Init()
	if err != nil {
		return nil, errors.Trace(err)
	}
	a.Bind(mux)

	return mux, nil
}
