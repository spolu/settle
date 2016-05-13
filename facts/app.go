package facts

import (
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/logging"
	"github.com/spolu/settl/util/respond"
	"goji.io"
)

// Build initializes the app and its web stack.
func Build() (*goji.Mux, error) {
	mux := goji.NewMux()
	mux.UseC(logging.RequestLogger)
	mux.UseC(respond.Recoverer)

	err := error(nil)

	f := &Configuration{}
	err = f.Init()
	if err != nil {
		return nil, errors.Trace(err)
	}
	f.Bind(mux)

	return mux, nil
}
