package api

import (
	"github.com/spolu/settl/api/lib/authentication"
	"github.com/spolu/settl/lib/errors"
	"github.com/spolu/settl/lib/livemode"
	"github.com/spolu/settl/lib/recoverer"
	"github.com/spolu/settl/lib/requestlogger"
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
