package main

import (
	"encoding/base64"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/logging"
)

// ethBackends map livemode to Ethereum backends.
var ethBackends = map[bool]backends.ContractBackend{}

func init() {
	liveClient, err := rpc.NewWSClient(os.Getenv("ETH_LIVE_WS_ENDPOINT"))
	if err != nil {
		log.Fatal(err)
	}
	ethBackends[true] = backends.NewRPCBackend(liveClient)

	testClient, err := rpc.NewWSClient(os.Getenv("ETH_TEST_WS_ENDPOINT"))
	if err != nil {
		log.Fatal(err)
	}
	ethBackends[false] = backends.NewRPCBackend(liveClient)
}

func main() {
	var lvm = flag.String("livemode", "false", "The livemode to use")
	var fct = flag.String("keyfile", "/dev/null", "The password protected key file")
	var pas = flag.String("password", "", "The password for the protected key")
	var chl = flag.String("challenge", "0", "The initial amount for the account")
	flag.Parse()

	ctx := context.Background()
	if *lvm == "true" {
		ctx = livemode.With(ctx, true)
	} else {
		ctx = livemode.With(ctx, false)
	}

	switch *fct {
	case "sign_challenge":
		signChallenge(ctx, *see, *chl)
	}
}

func signChallenge(
	ctx context.Context,
	keyfile string,
	passowrd string,
	challenge string,
) {
	logging.Logf(ctx,
		"Signing challenge: challenge=%q keyfile=%q",
		challenge, keyfile)

	keyjson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		log.Fatal(err)
	}
	key, err := accounts.DecryptKey(keyjson, auth)
	if err != nil {
		log.Fatal(err)
	}

	sig, err := crypto.Sign([]byte(challenge), key.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	logging.Logf(ctx, "Signed challenge: %s",
		base64.StdEncoding.EncodeToString([]byte(sig)))
}
