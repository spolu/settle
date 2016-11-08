package recoverer

import (
	"net/http"
	"runtime/debug"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/respond"
)

type middleware struct {
	http.Handler
}

// ServeHTTP handles incoming HTTP requests and attempt to recover panics.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	defer func() {
		if err := recover(); err != nil {
			logging.Logf(ctx, "---------------------")
			logging.Logf(ctx, "RECOVERER PRINT STACK")
			logging.Logf(ctx, "---------------------")
			debug.PrintStack()
			logging.Logf(ctx, "---------------------")
			if e, ok := err.(error); ok {
				logging.Logf(ctx, "Panic: error=%q", e.Error())
				respond.Error(ctx, w, errors.Trace(e))
			} else {
				logging.Logf(ctx, "Non error panic: dump=%+v", err)
				respond.Error(ctx, w, errors.Newf("Non error panic: %+v", err))
			}
		}
	}()

	m.Handler.ServeHTTP(w, r)
}

// Middleware that recovers from panics, logs the panic (and a backtrace), and
// returns a HTTP 500 (Internal Server Error) status if possible.
func Middleware(h http.Handler) http.Handler {
	return middleware{h}
}
