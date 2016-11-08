package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
	"goji.io"
)

var lvmFlag string
var usrFlag string
var pasFlag string
var actFlag string

func init() {
	flag.StringVar(&lvmFlag, "livemode", "false", "The livemode to use")
	flag.StringVar(&usrFlag, "username", "foo", "The user name of the user to upsert")
	flag.StringVar(&pasFlag, "password", "bar", "The password of the user to upsert")
	flag.StringVar(&actFlag, "action", "run", "The action to perform")

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
	if !flag.Parsed() {
		flag.Parse()
	}

	validActions := []string{"run", "create_user"}
	switch actFlag {
	case "run":
		mux, err := mint.Build()
		if err != nil {
			log.Fatal(errors.Details(err))
		}

		Serve(mux)

	case "create_user":
		ctx := context.Background()
		if lvmFlag == "true" {
			ctx = livemode.With(ctx, true)
		} else {
			ctx = livemode.With(ctx, false)
		}
		createUser(ctx, usrFlag, pasFlag)
	default:
		log.Fatalf("Invalid action `%s`, valid actions are: %s",
			actFlag, strings.Join(validActions, ", "))
	}
}

func createUser(
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
			log.Fatal(err)
		}
		err = user.Save(ctx)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		logging.Logf(ctx, "Creating user: %s", username)
		_, err := model.CreateUser(ctx, username, password)
		if err != nil {
			log.Fatal(err)
		}
	}
}
