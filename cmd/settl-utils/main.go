package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"golang.org/x/crypto/scrypt"

	"github.com/stellar/go-stellar-base/build"
	"github.com/stellar/go-stellar-base/horizon"
	"github.com/stellar/go-stellar-base/keypair"
)

func main() {
	var fct = flag.String("function", "sign", "The function to use")
	var see = flag.String("seed", "dummy", "The seed of the private key to sign with")
	var amt = flag.String("amount", "0", "The initial amount for the account")
	var chl = flag.String("challenge", "0", "The initial amount for the account")
	var adr = flag.String("address", "dummy", "The address of the fact to assert")
	var typ = flag.String("type", "dummy", "The type of the fact to assert")
	var val = flag.String("value", "", "The value of the fact to assert")
	flag.Parse()

	switch *fct {
	case "create_account":
		createAccount(*see, *amt)
	case "sign_challenge":
		signChallenge(*see, *chl)
	case "assert_fact":
		assertFact(*see, *adr, *typ, *val)
	case "deny_fact":
		denyFact(*see, *adr, *typ)
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

func assertFact(
	seed string,
	address string,
	typ string,
	value string,
) {
	fmt.Printf(
		"Creating fact: address=%q type=%q value=%q\n",
		address, typ, value)

	kp, err := keypair.Parse(seed)
	if err != nil {
		log.Fatal(err)
	}
	fkp := kp.(*keypair.Full)

	fmt.Printf("Fetching next sequence for fact verified account...\n")
	seq, err := horizon.DefaultPublicNetClient.SequenceForAccount(fkp.Address())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sequence: %d\n", seq)

	fmt.Printf("Generating scrypt of value...\n")
	scrypt, err := scrypt.Key([]byte(value), []byte(address), 16384, 8, 1, 64)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Scrypt (base64): %s\n", base64.StdEncoding.EncodeToString(scrypt))

	txEnvBuilder := build.TransactionEnvelopeBuilder{}
	txEnvBuilder.Init()
	txEnvBuilder.Mutate(
		build.Transaction(
			build.SourceAccount{fkp.Address()},
			build.Sequence{uint64(seq) + 1},
			build.SetData(
				fmt.Sprintf("fct.%s.%s", address, typ),
				scrypt,
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

func denyFact(
	seed string,
	address string,
	typ string,
) {
	fmt.Printf(
		"Denying fact: address=%q type=%q\n",
		address, typ)

	kp, err := keypair.Parse(seed)
	if err != nil {
		log.Fatal(err)
	}
	fkp := kp.(*keypair.Full)

	fmt.Printf("Fetching next sequence for fact verified account...\n")
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
			build.ClearData(
				fmt.Sprintf("fct.%s.%s", address, typ),
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
