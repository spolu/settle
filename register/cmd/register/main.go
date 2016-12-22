package main

import (
	"flag"
	"log"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/register/app"
)

var envFlag string

var hstFlag string
var prtFlag string

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

	flag.StringVar(&hstFlag, "host",
		"", "The host on which the register service is running")
	flag.StringVar(&prtFlag, "port",
		"", "The port on which the register service is running")

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

	if fl := log.Flags(); fl&log.Ltime != 0 {
		log.SetFlags(fl | log.Lmicroseconds)
	}
}

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}

	ctx, err := app.BackgroundContextFromFlags(
		envFlag,
		hstFlag, prtFlag,
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
	err = app.Serve(ctx, mux)
	if err != nil {
		log.Fatal(errors.Details(err))
	}
}
