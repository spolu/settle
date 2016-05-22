package recoverer

import (
	"net/http"
	"runtime/debug"

	"github.com/spolu/settl/lib/errors"
	"github.com/spolu/settl/lib/logging"
	"github.com/spolu/settl/lib/respond"

	"goji.io"
	"golang.org/x/net/context"
)

// Middleware that recovers from panics, logs the panic (and a backtrace), and
// returns a HTTP 500 (Internal Server Error) status if possible.
func Middleware(h goji.Handler) goji.Handler {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				if e, ok := err.(error); ok {
					logging.Logf(ctx, "Panic: error=%q", e.Error())
					respond.Error(ctx, w, errors.Trace(e))
				} else {
					logging.Logf(ctx, "Non error panic: dump=%+v", err)
					respond.Error(ctx, w, errors.Newf("Non error panic: %+v", err))
				}
				debug.PrintStack()
			}
		}()

		h.ServeHTTPC(ctx, w, r)
	}

	return goji.HandlerFunc(fn)
}
