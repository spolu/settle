package respond

import (
	"net/http"
	"runtime/debug"

	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/logging"

	"goji.io"
	"golang.org/x/net/context"
)

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible.
func Recoverer(h goji.Handler) goji.Handler {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				if e, ok := err.(error); ok {
					logging.Logf(ctx, "Panic: error=%q", e.Error())
					Error(ctx, w, errors.Trace(e))
				} else {
					logging.Logf(ctx, "Non error panic: dump=%+v", err)
					Error(ctx, w, errors.Newf("Non error panic: %+v", err))
				}
				debug.PrintStack()
			}
		}()

		h.ServeHTTPC(ctx, w, r)
	}

	return goji.HandlerFunc(fn)
}
