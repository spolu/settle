package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spolu/settle/lib/livemode"
	"github.com/spolu/settle/lib/logging"
)

// ethBackends map livemode to Ethereum backends.
var ethBackends = map[bool]bind.ContractBackend{}

func init() {
	fmt.Printf("Initializing...")
	liveClient, err := rpc.NewWSClient(os.Getenv("ETH_LIVE_WS_ENDPOINT"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("LV")
	ethBackends[true] = backends.NewRPCBackend(liveClient)

	testClient, err := rpc.NewWSClient(os.Getenv("ETH_TEST_WS_ENDPOINT"))
	if err != nil {
		log.Fatal(err)
	}
	ethBackends[false] = backends.NewRPCBackend(testClient)
}

func main() {
	var fct = flag.String("function", "sign_challenge", "the function to execute")
	var lvm = flag.String("livemode", "false", "The livemode to use")
	var key = flag.String("keyfile", "/dev/null", "The passphrase protected key file")
	var chl = flag.String("challenge", "0", "The initial amount for the account")
	var adr = flag.String("address", "", "The address to use")
	var val = flag.String("value", "", "The value to use")
	flag.Parse()

	ctx := context.Background()
	if *lvm == "true" {
		ctx = livemode.With(ctx, true)
	} else {
		ctx = livemode.With(ctx, false)
	}

	switch *fct {
	case "fact_hash":
		factHash(ctx, *adr, *val)
	case "sign_challenge":
		signChallenge(ctx, *key, *chl)
	}
}

func factHash(
	ctx context.Context,
	address string,
	value string,
) {
	logging.Logf(ctx, "Fact hash: %s",
		common.ToHex(
			crypto.Sha3(common.HexToAddress(address).Bytes(), []byte(value)),
		))
}

func signChallenge(
	ctx context.Context,
	keyfile string,
	challenge string,
) {
	logging.Logf(ctx,
		"Signing challenge: challenge=%q keyfile=%q",
		challenge, keyfile)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Passphrase: ")
	passphrase, _ := reader.ReadString('\n')
	passphrase = passphrase[:len(passphrase)-1]

	keyjson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		log.Fatal(err)
	}
	key, err := accounts.DecryptKey(keyjson, passphrase)
	if err != nil {
		log.Fatal(err)
	}

	sig, err := crypto.Sign(crypto.Sha3([]byte(challenge)), key.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	logging.Logf(ctx, "Signed challenge: %s",
		base64.StdEncoding.EncodeToString([]byte(sig)))
}
