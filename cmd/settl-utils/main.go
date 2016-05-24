package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"github.com/spolu/settl/facts"
	"github.com/stellar/go-stellar-base/build"
	"github.com/stellar/go-stellar-base/horizon"
	"github.com/stellar/go-stellar-base/keypair"
)

// clients maps livemodes to Horizon clients.
var clients = map[bool]*horizon.Client{
	false: horizon.DefaultTestNetClient,
	true:  horizon.DefaultPublicNetClient,
}

func main() {
	var fct = flag.String("function", "sign", "The function to use")
	var see = flag.String("seed", "dummy", "The seed of the private key to sign with")
	var amt = flag.String("amount", "0", "The initial amount for the account")
	var chl = flag.String("challenge", "0", "The initial amount for the account")
	var adr = flag.String("address", "dummy", "The address of the fact to assert")
	var typ = flag.String("type", "dummy", "The type of the fact to assert")
	var val = flag.String("value", "", "The value of the fact to assert")
	var lvm = flag.String("livemode", "false", "The livemode to use")
	flag.Parse()

	livemode := false
	if *lvm == "true" {
		livemode = true
	}

	switch *fct {
	case "sign_challenge":
		signChallenge(*see, *chl)
	case "create_account":
		createAccount(livemode, *see, *amt)
	case "assert_fact":
		assertFact(livemode, *see, *adr, *typ, *val)
	case "revoke_fact":
		revokeFact(livemode, *see, *adr, *typ)
	}
}

func signChallenge(
	seed string,
	challenge string,
) {
	fmt.Printf(
		"Signing challenge: challenge=%q\n",
		challenge)

	kp, err := keypair.Parse(seed)
	if err != nil {
		log.Fatal(err)
	}
	fkp := kp.(*keypair.Full)

	sign, err := fkp.Sign([]byte(challenge))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Signed challenge: %s\n",
		base64.StdEncoding.EncodeToString([]byte(sign)))
}

func createAccount(
	livemode bool,
	seed string,
	amount string,
) {
	fmt.Printf(
		"Creating Stellar Account: livemode=%t, amount=%q\n",
		livemode, amount)

	kp, err := keypair.Parse(seed)
	if err != nil {
		log.Fatal(err)
	}
	fkp := kp.(*keypair.Full)

	fmt.Printf("Generating new KeyPair...\n")
	nk, err := keypair.Random()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Seed: %s\n", nk.Seed())
	fmt.Printf("Address: %s\n", nk.Address())

	fmt.Printf("Fetching next sequence for creator account...\n")
	seq, err := clients[livemode].SequenceForAccount(fkp.Address())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sequence: %d\n", seq)

	buildNetwork := build.TestNetwork
	if livemode {
		buildNetwork = build.PublicNetwork
	}

	txEnvBuilder := build.TransactionEnvelopeBuilder{}
	txEnvBuilder.Init()
	txEnvBuilder.Mutate(
		build.Transaction(
			build.SourceAccount{fkp.Address()},
			build.Sequence{uint64(seq) + 1},
			build.CreateAccount(
				build.NativeAmount{amount},
				build.Destination{nk.Address()},
			),
			buildNetwork,
		),
		build.Sign{fkp.Seed()},
	)

	if txEnvBuilder.Err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Generating envelope...\n")
	env, err := txEnvBuilder.Base64()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Envelope: %s\n", env)

	fmt.Printf("Submitting transaction...\n")
	res, err := clients[livemode].SubmitTransaction(env)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		switch err := err.(type) {
		case *horizon.Error:
			fmt.Printf("Problem: %+v\n", err.Problem)
		}
	} else {
		fmt.Printf("Response: %+v\n", res)
	}
}

func assertFact(
	livemode bool,
	seed string,
	address string,
	typ string,
	value string,
) {
	fmt.Printf(
		"Creating fact: livemode=%t address=%q type=%q value=%q\n",
		livemode, address, typ, value)

	assertion, err := facts.AssertFact(
		livemode, seed, address, facts.FctType(typ), value)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		switch err := err.(type) {
		case *horizon.Error:
			fmt.Printf("Problem: %+v\n", err.Problem)
		}
	} else {
		fmt.Printf("Transaction Hash: %s\n", assertion.TransactionHash)
	}
}

func revokeFact(
	livemode bool,
	seed string,
	address string,
	typ string,
) {
	fmt.Printf(
		"Revoking fact: livemode=%t address=%q type=%q\n",
		livemode, address, typ)

	revocation, err := facts.RevokeFact(
		livemode, seed, address, facts.FctType(typ))
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		switch err := err.(type) {
		case *horizon.Error:
			fmt.Printf("Problem: %+v\n", err.Problem)
		}
	} else {
		fmt.Printf("Transaction Hash: %s\n", revocation.TransactionHash)
	}
}
