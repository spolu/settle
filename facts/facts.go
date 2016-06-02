package facts

import (
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/net/context"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/livemode"
	"github.com/stellar/go-stellar-base/build"
	"github.com/stellar/go-stellar-base/horizon"
	"github.com/stellar/go-stellar-base/keypair"
)

// FctType are the possible standard fact types.
type FctType string

const (
	// FctName is the full name of the individual, company or organization.
	FctName FctType = "name"
	// FctEntityType is the entity type (`individual`, `for-profit`, `non-profit`,
	//`state`).
	FctEntityType FctType = "entity_type"
	// FctDateOfBirth is the date of birth of an individual (YYYY-MM-DD)
	FctDateOfBirth FctType = "date_of_birth"
	// FctDateOfCreation is the date of creation of an organization
	// (YYYY-MM-DD).
	FctDateOfCreation FctType = "date_of_creation"
	// FctDateOfIncorporation is the date of creation of a company
	// (YYYY-MM-DD).
	FctDateOfIncorporation FctType = "date_of_incorporation"
	// FctEmail is the fully qualified lowercased email address.
	FctEmail FctType = "email"
	// FctPhone is the fully qualified phone number without space or sperator
	// and starting with `+` and country code.
	FctPhone FctType = "phone"
	// FctURL is a fully qualifed URL owned by the entity.
	FctURL FctType = "url"
)

// factTypeToCode maps FctType to their type code.
var factTypeToCode = map[FctType]string{
	FctName:                "000",
	FctEntityType:          "001",
	FctDateOfBirth:         "002",
	FctDateOfCreation:      "002",
	FctDateOfIncorporation: "002",
	FctEmail:               "010",
	FctPhone:               "011",
	FctURL:                 "012",
}

// Assertion is the object returned when a fact is successfully asserted.
type Assertion struct {
	Livemode        bool
	Address         string
	Type            FctType
	Value           string
	Verifier        string
	TransactionHash string
}

// Revocation is the object returned when a fact is successfully revoked.
type Revocation struct {
	Livemode        bool
	Address         string
	Type            FctType
	Verifier        string
	TransactionHash string
}

// clients maps livemodes to Horizon clients.
var clients = map[bool]*horizon.Client{
	false: horizon.DefaultTestNetClient,
	true:  horizon.DefaultPublicNetClient,
}

// CheckFact checks that the fact (address, typ, value) is asserted by the
// verifier on the test network if livemode is false, and the live network
// otherwise.
func CheckFact(
	ctx context.Context,
	address string,
	typ FctType,
	value string,
	verifier string,
) error {
	code, ok := factTypeToCode[typ]
	if !ok {
		return errors.Newf("Invalid fact type: %s", typ)
	}
	key := fmt.Sprintf("fct.%s.%s", address, code)

	scrypt, err := scrypt.Key([]byte(value), []byte(verifier), 16384, 8, 1, 64)
	if err != nil {
		return errors.Trace(err)
	}

	account, err := clients[livemode.Get(ctx)].LoadAccount(verifier)
	if err != nil {
		return errors.Trace(err)
	}

	check, ok := account.Data[key]
	if !ok || check != base64.StdEncoding.EncodeToString(scrypt) {
		return errors.Newf("The verifier does not assert this fact")
	}

	return nil
}

// AssertFact attempts to assert a fact using the seed provided as argument as
// verifier.
func AssertFact(
	ctx context.Context,
	seed string,
	address string,
	typ FctType,
	value string,
) (*Assertion, error) {
	code, ok := factTypeToCode[typ]
	if !ok {
		return nil, errors.Newf("Invalid fact type: %s", typ)
	}
	key := fmt.Sprintf("fct.%s.%s", address, code)

	kp, err := keypair.Parse(seed)
	if err != nil {
		return nil, errors.Trace(err)
	}
	fkp := kp.(*keypair.Full)

	seq, err := clients[livemode.Get(ctx)].SequenceForAccount(fkp.Address())
	if err != nil {
		return nil, errors.Trace(err)
	}

	scrypt, err := scrypt.Key([]byte(value), []byte(fkp.Address()), 16384, 8, 1, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}

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
			build.SetData(key, scrypt),
			buildNetwork,
		),
		build.Sign{fkp.Seed()},
	)
	if txEnvBuilder.Err != nil {
		return nil, errors.Trace(txEnvBuilder.Err)
	}
	env, err := txEnvBuilder.Base64()
	if err != nil {
		return nil, errors.Trace(err)
	}

	res, err := clients[livemode.Get(ctx)].SubmitTransaction(env)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Assertion{
		Livemode:        livemode.Get(ctx),
		Address:         address,
		Type:            typ,
		Value:           value,
		Verifier:        fkp.Address(),
		TransactionHash: res.Hash,
	}, nil
}

// RevokeFact attempts to revoke a fact using the seed provided as argument as
// verifier.
func RevokeFact(
	ctx context.Context,
	seed string,
	address string,
	typ FctType,
) (*Revocation, error) {
	code, ok := factTypeToCode[typ]
	if !ok {
		return nil, errors.Newf("Invalid fact type: %s", typ)
	}
	key := fmt.Sprintf("fct.%s.%s", address, code)

	kp, err := keypair.Parse(seed)
	if err != nil {
		return nil, errors.Trace(err)
	}
	fkp := kp.(*keypair.Full)

	seq, err := clients[livemode.Get(ctx)].SequenceForAccount(fkp.Address())
	if err != nil {
		return nil, errors.Trace(err)
	}

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
			build.ClearData(key),
			buildNetwork,
		),
		build.Sign{fkp.Seed()},
	)
	if txEnvBuilder.Err != nil {
		return nil, errors.Trace(txEnvBuilder.Err)
	}
	env, err := txEnvBuilder.Base64()
	if err != nil {
		return nil, errors.Trace(err)
	}

	res, err := clients[livemode.Get(ctx)].SubmitTransaction(env)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Revocation{
		Livemode:        livemode.Get(ctx),
		Address:         address,
		Type:            typ,
		Verifier:        fkp.Address(),
		TransactionHash: res.Hash,
	}, nil
}
