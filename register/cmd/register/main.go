package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	goji "goji.io"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/register/app"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
)

var envFlag string

var dsnFlag string
var crdFlag string

var mntFlag string
var mdsFlag string

var smlFlag string
var smpFlag string
var smhFlag string

var rcpFlag string

func init() {
	flag.StringVar(&envFlag, "env",
		"qa", "The environment to run in (qa, production), default: qa")

	flag.StringVar(&dsnFlag, "db_dsn",
		"", "The DSN of the database to use, default: sqlite3://~/.mint/register-$env.db")
	flag.StringVar(&dsnFlag, "credentials_url",
		"", "The URL users receive over email to retrieve their credentials")

	flag.StringVar(&mntFlag, "mint",
		"", "The mint this register service is bound to")
	flag.StringVar(&mdsFlag, "mint_dsn",
		"", "The DSN of the mint database to use, default: sqlite3://~/.mint/mint-$env.db")

	flag.StringVar(&rcpFlag, "recaptcha_secret",
		"", "The reCAPTCHA secret to use to verify users")

	flag.StringVar(&smlFlag, "smtp_login",
		"", "The SMTP login to use to send verification emails")
	flag.StringVar(&smpFlag, "smtp_password",
		"", "The SMTP password to use to send verification emails")
	flag.StringVar(&smhFlag, "smtp_host",
		"", "The SMTP host to use to send verification emails, including the port")

	bind.WithFlag()
	if fl := log.Flags(); fl&log.Ltime != 0 {
		log.SetFlags(fl | log.Lmicroseconds)
	}
	graceful.DoubleKickWindow(2 * time.Second)
}

// Serve starts the given mux using reasonable defaults.
func Serve(mux *goji.Mux) {
	ServeListener(mux, bind.Default())
}

// ServeListener is like Serve, but runs `mux` on top of an arbitrary
// net.Listener.
func ServeListener(mux *goji.Mux, listener net.Listener) {
	// Install our handler at the root of the standard net/http default mux.
	// This allows packages like expvar to continue working as expected.
	http.Handle("/", mux)

	log.Println("Starting on", listener.Addr())

	graceful.HandleSignals()
	bind.Ready()
	graceful.PreHook(func() { log.Printf("Received signal, gracefully stopping") })
	graceful.PostHook(func() { log.Printf("Stopped") })

	err := graceful.Serve(listener, http.DefaultServeMux)

	if err != nil {
		log.Fatal(err)
	}

	graceful.Wait()
}

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}

	ctx, err := app.BackgroundContextFromFlags(
		envFlag,
		dsnFlag, crdFlag,
		mntFlag, mdsFlag,
		smlFlag, smpFlag, smhFlag,
		rcpFlag,
	)
	if err != nil {
		log.Fatal(errors.Details(err))
	}

	mux, err := app.Build(ctx)
	if err != nil {
		log.Fatal(errors.Details(err))
	}
	Serve(mux)
}
