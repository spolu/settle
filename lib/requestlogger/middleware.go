package requestlogger

import (
	"log"
	"net/http"
	"time"

	"github.com/spolu/settle/lib/logging"
	"github.com/zenazn/goji/web/mutil"

	"goji.io"

	"golang.org/x/net/context"
)

func init() {
	log.SetFlags(0)
}

type middleware struct {
	goji.Handler
}

// ServeHTTPC handles incoming HTTP requests and attempt to log them.
func (m middleware) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	start := time.Now()
	url := *r.URL
	wp := mutil.WrapWriter(w)

	logging.Logf(ctx, "HTTP Request: method=%q url=%q remote=%q",
		r.Method, url.String(), r.RemoteAddr)

	defer func() {
		wp.WriteHeader(http.StatusOK)
		logging.Logf(ctx, "HTTP Response: status=%d latency=%d",
			wp.Status(), time.Now().Sub(start)/time.Millisecond)
	}()

	m.Handler.ServeHTTPC(ctx, wp, r)
}

// Middleware that logs methods, URLs, remote addresses, status, lantency.
func Middleware(h goji.Handler) goji.Handler {
	return middleware{h}
}
