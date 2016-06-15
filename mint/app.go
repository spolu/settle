package mint

import (
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/recoverer"
	"github.com/spolu/settle/lib/requestlogger"
	"github.com/spolu/settle/mint/lib/authentication"
	"goji.io"
)

// Build initializes the app and its web stack.
func Build() (*goji.Mux, error) {
	mux := goji.NewMux()
	mux.UseC(recoverer.Middleware)
	mux.UseC(requestlogger.Middleware)
	mux.UseC(livemode.Middleware)
	mux.UseC(authentication.Middleware)

	err := error(nil)

	a := &Configuration{}
	err = a.Init()
	if err != nil {
		return nil, errors.Trace(err)
	}
	a.Bind(mux)

	return mux, nil
}
