package logging

import (
	"log"
	"net/http"
	"time"

	"github.com/zenazn/goji/web/mutil"

	"goji.io"

	"golang.org/x/net/context"
)

func init() {
	log.SetFlags(0)
}

type requestLogger struct {
	goji.Handler
}

// ServeHTTPC handles incoming HTTP requests and attempt to log them.
func (rl requestLogger) ServeHTTPC(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	start := time.Now()
	url := *r.URL
	wp := mutil.WrapWriter(w)

	Logf(ctx, "Request: method=%q url=%q remote=%q",
		r.Method, url.String(), r.RemoteAddr, wp.Status())

	defer func() {
		wp.WriteHeader(http.StatusOK)
		Logf(ctx, "Response: status=%d latency=%d",
			wp.Status(), time.Now().Sub(start)/time.Millisecond)
	}()

	rl.Handler.ServeHTTPC(ctx, wp, r)
}

// RequestLogger is a middleware that logs URLs, headers, status, lantency.
func RequestLogger(h goji.Handler) goji.Handler {
	return requestLogger{h}
}
