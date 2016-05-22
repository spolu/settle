package authentication

import (
	"log"
	"os"

	"github.com/spolu/settl/lib/errors"
	"github.com/stellar/go-stellar-base/keypair"
)

var rootTestSeed string
var rootLiveSeed string

// RootTestKeypair is the root keypair to use in test mode.
var RootTestKeypair *keypair.Full

// RootLiveKeypair is the root keypair to use in live mode.
var RootLiveKeypair *keypair.Full

func init() {
	rootTestSeed = os.Getenv("ROOT_TEST_SEED")
	rootLiveSeed = os.Getenv("ROOT_LIVE_SEED")

	tkp, err := keypair.Parse(rootTestSeed)
	if err != nil {
		log.Fatal(errors.Details(err))
	}
	lkp, err := keypair.Parse(rootLiveSeed)
	if err != nil {
		log.Fatal(errors.Details(err))
	}

	RootTestKeypair = tkp.(*keypair.Full)
	RootLiveKeypair = lkp.(*keypair.Full)
}
