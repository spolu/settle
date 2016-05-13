package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/stellar/go-stellar-base/build"
	"github.com/stellar/go-stellar-base/horizon"
	"github.com/stellar/go-stellar-base/keypair"
	"github.com/stellar/go-stellar-base/xdr"
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

	fmt.Printf("Generating new KeyPair...\n")
	nk, err := keypair.Random()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Seed: %s\n", nk.Seed())
	fmt.Printf("Address: %s\n", nk.Address())

	fmt.Printf("Fetching next sequence for creator account...\n")
	seq, err := horizon.DefaultPublicNetClient.SequenceForAccount(kp.Address())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sequence: %d\n", seq)

	txEnvBuilder := build.TransactionEnvelopeBuilder{}
	txEnvBuilder.Init()
	txEnvBuilder.Mutate(
		build.Transaction(
			build.SourceAccount{kp.Address()},
			build.Sequence{uint64(seq)},
			build.CreateAccount(
				build.NativeAmount{*amt},
				build.SourceAccount{kp.Address()},
				build.Destination{nk.Address()},
			),
		),
		build.Sign{*see},
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

	var rawEnv xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(env, &rawEnv)
	if err != nil {
		log.Fatal(err)
	}
	chk, err := xdr.MarshalBase64(rawEnv)
	if err != nil {
		log.Fatal(err)
	}
	if chk != env {
		log.Fatal(fmt.Errorf("Mismatch!"))
	}

	fmt.Printf("Submitting transaction...\n")
	res, err := horizon.DefaultPublicNetClient.SubmitTransaction(env)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response: %+v\n", res)
}
