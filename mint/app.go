package mint

import (
	"os"

	"github.com/spolu/peer-currencies/lib/errors"
	"github.com/spolu/peer-currencies/lib/livemode"
	"github.com/spolu/peer-currencies/lib/recoverer"
	"github.com/spolu/peer-currencies/lib/requestlogger"
	"github.com/spolu/peer-currencies/mint/lib/authentication"
	"goji.io"

	// force initialization of schemas
	_ "github.com/spolu/peer-currencies/model/schemas"
)

// Build initializes the app and its web stack.
func Build() (*goji.Mux, error) {
	mux := goji.NewMux()
	mux.UseC(recoverer.Middleware)
	mux.UseC(requestlogger.Middleware)
	mux.UseC(livemode.Middleware)
	mux.UseC(authentication.Middleware)

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
