package requestlogger

import (
	"log"
	"net/http"
	"time"

	"github.com/spolu/settle/lib/logging"
	"github.com/zenazn/goji/web/mutil"
)

func init() {
	log.SetFlags(0)
}

type middleware struct {
	http.Handler
}

// ServeHTTP handles incoming HTTP requests and attempt to log them.
func (m middleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	start := time.Now()
	url := *r.URL
	wp := mutil.WrapWriter(w)

	logging.Logf(ctx, "HTTP Request: method=%q path=%q remote=%q",
		r.Method, url.RequestURI(), r.RemoteAddr)

	defer func() {
		wp.WriteHeader(http.StatusOK)
		logging.Logf(ctx, "HTTP Response: status=%d latency=%d",
			wp.Status(), time.Now().Sub(start)/time.Millisecond)
	}()

	m.Handler.ServeHTTP(wp, r)
}

// Middleware that logs methods, URLs, remote addresses, status, lantency.
func Middleware(h http.Handler) http.Handler {
	return middleware{h}
}
