package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/mint/app"
	"github.com/spolu/settle/mint/model"
)

var actFlag string

var envFlag string
var dsnFlag string

var hstFlag string
var prtFlag string

var usrFlag string
var pasFlag string

func init() {
	flag.StringVar(&actFlag, "action",
		"run", "The action to perform (run, create_user), default: run")

	flag.StringVar(&envFlag, "env",
		"qa", "The environment to run in (qa, production), default: qa")
	flag.StringVar(&dsnFlag, "db_dsn",
		"", "The DSN of the database to use, default: sqlite3://~/.mint/mint-$env.db")
	flag.StringVar(&hstFlag, "host",
		"", "The externally accessible host name of this mint, default: none (required for production)")
	flag.StringVar(&prtFlag, "port",
		"", "The port on which the mint will listen, default: 2406 in qa and 2407 in production")

	flag.StringVar(&usrFlag, "username",
		"foo", "The user name of the user for the create_user action")
	flag.StringVar(&pasFlag, "password",
		"bar", "The password of the user for the create_user action")

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
		dsnFlag,
		hstFlag, prtFlag,
	)
	if err != nil {
		log.Fatal(errors.Details(err))
	}

	validActions := []string{"run", "create_user"}
	switch actFlag {
	case "run":
		mux, err := app.Build(ctx)
		if err != nil {
			log.Fatal(errors.Details(err))
		}
		err = app.Serve(ctx, mux)
		if err != nil {
			log.Fatal(errors.Details(err))
		}
	case "create_user":
		CreateUser(ctx, usrFlag, pasFlag)
	default:
		log.Fatalf("Invalid action `%s`, valid actions are: %s",
			actFlag, strings.Join(validActions, ", "))
	}
}

// CreateUser is a convenience function exposed on the command line
func CreateUser(
	ctx context.Context,
	username string,
	password string,
) {
	user, err := model.LoadUserByUsername(ctx, username)
	if err != nil {
		log.Fatal(err)
	}

	if user != nil {
		logging.Logf(ctx, "Updating user: %s", username)
		err := user.UpdatePassword(ctx, password)
		if err != nil {
			log.Fatal(errors.Details(err))
		}
		err = user.Save(ctx)
		if err != nil {
			log.Fatal(errors.Details(err))
		}
	} else {
		logging.Logf(ctx, "Creating user: %s", username)
		_, err := model.CreateUser(ctx, username, password)
		if err != nil {
			log.Fatal(errors.Details(err))
		}
	}
}
