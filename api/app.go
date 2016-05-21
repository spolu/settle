package api

import (
	"github.com/spolu/settl/api/util/auth"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/logging"
	"github.com/spolu/settl/util/respond"
	"goji.io"
)

// Build initializes the app and its web stack.
func Build() (*goji.Mux, error) {
	mux := goji.NewMux()
	mux.UseC(respond.Recoverer)
	mux.UseC(logging.RequestLogger)
	mux.UseC(auth.Authenticator)

	err := error(nil)

	a := &Configuration{}
	err = a.Init()
	if err != nil {
		return nil, errors.Trace(err)
	}
	a.Bind(mux)

	return mux, nil
}
