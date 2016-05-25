package main

import (
	"encoding/base64"
	"flag"
	"log"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/net/context"

	"github.com/spolu/settl/facts"
	"github.com/spolu/settl/lib/livemode"
	"github.com/spolu/settl/lib/logging"
	"github.com/spolu/settl/lib/xor"
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
	var lvm = flag.String("livemode", "false", "The livemode to use")
	var fct = flag.String("function", "sign", "The function to use")
	var see = flag.String("seed", "dummy", "The seed of the private key to sign with")
	var amt = flag.String("amount", "0", "The initial amount for the account")
	var chl = flag.String("challenge", "0", "The initial amount for the account")
	var adr = flag.String("address", "dummy", "The address of the fact to assert")
	var typ = flag.String("type", "dummy", "The type of the fact to assert")
	var val = flag.String("value", "", "The value of the fact to assert")
	var pas = flag.String("password", "", "The password for scrypt")
	var enc = flag.String("encrypted", "", "The the encrypted seed")
	flag.Parse()

	ctx := context.Background()
	if *lvm == "true" {
		ctx = livemode.With(ctx, true)
	} else {
		ctx = livemode.With(ctx, false)
	}

	switch *fct {
	case "create_stellar_account":
		createStellarAccount(ctx, *see, *amt)
	case "sign_challenge":
		signChallenge(ctx, *see, *chl)
	case "assert_fact":
		assertFact(ctx, *see, *adr, *typ, *val)
	case "revoke_fact":
		revokeFact(ctx, *see, *adr, *typ)
	case "encrypt_seed":
		encryptSeed(ctx, *see, *pas)
	case "decrypt_seed":
		decryptSeed(ctx, *adr, *enc, *pas)
	}
}

func signChallenge(
	ctx context.Context,
	seed string,
	challenge string,
) {
	logging.Logf(ctx,
		"Signing challenge: challenge=%q",
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
	logging.Logf(ctx, "Signed challenge: %s",
		base64.StdEncoding.EncodeToString([]byte(sign)))
}

func createStellarAccount(
	ctx context.Context,
	seed string,
	amount string,
) {
	logging.Logf(ctx,
		"Creating Stellar Account: livemode=%t, amount=%q",
		livemode.Get(ctx), amount)

	kp, err := keypair.Parse(seed)
	if err != nil {
		log.Fatal(err)
	}
	fkp := kp.(*keypair.Full)

	logging.Logf(ctx, "Generating new KeyPair...")
	nk, err := keypair.Random()
	if err != nil {
		log.Fatal(err)
	}
	logging.Logf(ctx, "Seed: %s", nk.Seed())
	logging.Logf(ctx, "Address: %s", nk.Address())

	logging.Logf(ctx, "Fetching next sequence for creator account...")
	seq, err := clients[livemode.Get(ctx)].SequenceForAccount(fkp.Address())
	if err != nil {
		log.Fatal(err)
	}
	logging.Logf(ctx, "Sequence: %d", seq)

	buildNetwork := build.TestNetwork
	if livemode.Get(ctx) {
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

	logging.Logf(ctx, "Generating envelope...")
	env, err := txEnvBuilder.Base64()
	if err != nil {
		log.Fatal(err)
	}
	logging.Logf(ctx, "Envelope: %s", env)

	logging.Logf(ctx, "Submitting transaction...")
	res, err := clients[livemode.Get(ctx)].SubmitTransaction(env)
	if err != nil {
		logging.Logf(ctx, "Error: %+v", err)
		switch err := err.(type) {
		case *horizon.Error:
			logging.Logf(ctx, "Problem: %+v", err.Problem)
		}
	} else {
		logging.Logf(ctx, "Response: %+v", res)
	}
}

func assertFact(
	ctx context.Context,
	seed string,
	address string,
	typ string,
	value string,
) {
	logging.Logf(ctx,
		"Creating fact: livemode=%t address=%q type=%q value=%q",
		livemode.Get(ctx), address, typ, value)

	assertion, err := facts.AssertFact(
		ctx, seed, address, facts.FctType(typ), value)
	if err != nil {
		logging.Logf(ctx, "Error: %+v", err)
		switch err := err.(type) {
		case *horizon.Error:
			logging.Logf(ctx, "Problem: %+v", err.Problem)
		}
	} else {
		logging.Logf(ctx, "Transaction Hash: %s", assertion.TransactionHash)
	}
}

func revokeFact(
	ctx context.Context,
	seed string,
	address string,
	typ string,
) {
	logging.Logf(ctx,
		"Revoking fact: livemode=%t address=%q type=%q",
		livemode.Get(ctx), address, typ)

	revocation, err := facts.RevokeFact(
		ctx, seed, address, facts.FctType(typ))
	if err != nil {
		logging.Logf(ctx, "Error: %+v", err)
		switch err := err.(type) {
		case *horizon.Error:
			logging.Logf(ctx, "Problem: %+v", err.Problem)
		}
	} else {
		logging.Logf(ctx, "Transaction Hash: %s", revocation.TransactionHash)
	}
}

func encryptSeed(
	ctx context.Context,
	seed string,
	password string,
) {
	kp, err := keypair.Parse(seed)
	if err != nil {
		log.Fatal(err)
	}
	fkp := kp.(*keypair.Full)

	logging.Logf(ctx,
		"Scrypting seed: livemode=%t",
		livemode.Get(ctx))

	scrypt, err := scrypt.Key(
		[]byte(password), []byte(fkp.Address()), 16384, 8, 1, len(seed))
	if err != nil {
		log.Fatal(err)
	}

	encryptedSeed := make([]byte, len(seed))
	xor.Bytes(encryptedSeed, scrypt, []byte(seed))

	logging.Logf(ctx, "Scrypted seed: %s (%d bytes)",
		base64.StdEncoding.EncodeToString([]byte(encryptedSeed)), len(seed))
}

func decryptSeed(
	ctx context.Context,
	address string,
	encrypted string,
	password string,
) {
	logging.Logf(ctx,
		"Decrypting seed: livemode=%t address=%q encrypted=%q",
		livemode.Get(ctx), address, encrypted)

	bytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		log.Fatal(err)
	}

	scrypt, err := scrypt.Key(
		[]byte(password), []byte(address), 16384, 8, 1, len(bytes))
	if err != nil {
		log.Fatal(err)
	}

	decryptedSeed := make([]byte, len(bytes))
	xor.Bytes(decryptedSeed, scrypt, bytes)

	logging.Logf(ctx, "Decrypted seed: %s (%d bytes)",
		string(decryptedSeed), len(bytes))
}
