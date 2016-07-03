package main

import (
	"flag"
	"log"

	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/spolu/peer-currencies/lib/livemode"
	"github.com/spolu/peer-currencies/lib/logging"
	"github.com/spolu/peer-currencies/model"
)

// ethBackends map livemode to Ethereum backends.
var ethBackends = map[bool]bind.ContractBackend{}

func main() {
	var fct = flag.String("function", "none", "the function to execute")
	var lvm = flag.String("livemode", "false", "The livemode to use")
	var usr = flag.String("username", "foo", "The user name of the user to upsert")
	var pas = flag.String("password", "bar", "The password of the user to upsert")
	flag.Parse()

	ctx := context.Background()
	if *lvm == "true" {
		ctx = livemode.With(ctx, true)
	} else {
		ctx = livemode.With(ctx, false)
	}

	switch *fct {
	case "upsert_user":
		upsertUser(ctx, *usr, *pas)
	}
}

func upsertUser(
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
