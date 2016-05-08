package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"github.com/spolu/settl/model"
	"github.com/stellar/go-stellar-base/keypair"
)

func main() {
	var act = flag.String("action", "", "Action to sign: assert, revoke")
	var acc = flag.String("account", "", "The fact account")
	var typ = flag.String("type", "", "The fact type")
	var val = flag.String("value", "", "The fact velue")
	var see = flag.String("seed", "", "The sede of the private key to sign with")
	flag.Parse()

	fmt.Printf(
		"Fact: account=%q type=%q value=%q\n",
		*acc, *typ, *val)

	f := model.NewFact(
		model.PublicKey(*acc),
		model.FctType(*typ),
		*val,
	)
	payload := f.PayloadForAction(model.FctAction(*act))

	fmt.Printf(
		"Payload: action=%q payload=%q\n",
		*act, payload)

	kp, err := keypair.Parse(*see)
	if err != nil {
		log.Fatal(err)
	}

	b, err := kp.Sign([]byte(payload))
	if err != nil {
		log.Fatal(err)
	}

	sig := base64.StdEncoding.EncodeToString(b)
	fmt.Printf(
		"Signature: signature=%q\n",
		string(sig))
}
