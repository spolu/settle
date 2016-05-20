package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"github.com/stellar/go-stellar-base/build"
	"github.com/stellar/go-stellar-base/horizon"
	"github.com/stellar/go-stellar-base/keypair"
)

func main() {
	var fct = flag.String("function", "sign", "The function to use")
	var see = flag.String("seed", "dummy", "The seed of the private key to sign with")
	var amt = flag.String("amount", "0", "The initial amount for the account")
	var chl = flag.String("challenge", "0", "The initial amount for the account")
	flag.Parse()

	switch *fct {
	case "create_account":
		createAccount(*see, *amt)
	case "sign_challenge":
		signChallenge(*see, *chl)
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
	seed string,
	amount string,
) {
	fmt.Printf(
		"Creating Stellar Account: amount=%q\n",
		amount)

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
	seq, err := horizon.DefaultPublicNetClient.SequenceForAccount(fkp.Address())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sequence: %d\n", seq)

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
			build.PublicNetwork,
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
	res, err := horizon.DefaultPublicNetClient.SubmitTransaction(env)
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
