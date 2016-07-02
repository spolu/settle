package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"goji.io"

	"github.com/spolu/peer_currencies/lib/errors"
	"github.com/spolu/peer_currencies/mint"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
)

func init() {
	bind.WithFlag()
	if fl := log.Flags(); fl&log.Ltime != 0 {
		log.SetFlags(fl | log.Lmicroseconds)
	}
	graceful.DoubleKickWindow(2 * time.Second)
}

// Serve starts the given mux using reasonable defaults.
func Serve(mux *goji.Mux) {
	if !flag.Parsed() {
		flag.Parse()
	}

	ServeListener(mux, bind.Default())
}

// ServeListener is like Serve, but runs `mux` on top of an arbitrary
// net.Listener.
func ServeListener(mux *goji.Mux, listener net.Listener) {
	// Install our handler at the root of the standard net/http default mux.
	// This allows packages like expvar to continue working as expected.
	http.Handle("/", mux)

	log.Println("Starting Goji on", listener.Addr())

	graceful.HandleSignals()
	bind.Ready()
	graceful.PreHook(func() { log.Printf("Goji received signal, gracefully stopping") })
	graceful.PostHook(func() { log.Printf("Goji stopped") })

	err := graceful.Serve(listener, http.DefaultServeMux)

	if err != nil {
		log.Fatal(err)
	}

	graceful.Wait()
}

func main() {
	mux, err := mint.Build()
	if err != nil {
		log.Fatal(errors.Details(err))
	}

	Serve(mux)
}
