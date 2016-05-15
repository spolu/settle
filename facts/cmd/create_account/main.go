package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/stellar/go-stellar-base/build"
	"github.com/stellar/go-stellar-base/horizon"
	"github.com/stellar/go-stellar-base/keypair"
)

func main() {
	var amt = flag.String("amount", "0", "The initial amount for the account")
	var see = flag.String("seed", "dummy", "The seed of the private key to sign with")
	flag.Parse()

	fmt.Printf(
		"Creating Stellar Account: amount=%q\n",
		*amt)

	kp, err := keypair.Parse(*see)
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
				build.NativeAmount{*amt},
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
