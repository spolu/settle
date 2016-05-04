package main

import (
	"fmt"
	"log"

	"github.com/spolu/settl/model"
)

func main() {
	f := model.NewFact(
		model.PublicKey("TESTENTITY"),
		model.FctName,
		"Stanislas Polu",
	)
	err := f.Save()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved fact: %+v\n", f)

	f, err = model.LoadFact(f.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved fact: %+v\n", f)

	s := model.NewAssertion(
		f.ID,
		model.PublicKey("TESTENTITY"),
		"TEST_SIGNATURE_SIG",
	)
	err = s.Save()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved assertion: %+v\n", s)

	s, err = model.LoadAssertion(s.ID, s.Fact)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved assertion: %+v\n", s)

	r := model.NewRevocation(
		f.ID,
		model.PublicKey("TESTENTITY"),
		"TEST_REVOCATION_SIG",
	)
	err = r.Save()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved revocation: %+v\n", r)

	r, err = model.LoadRevocation(r.ID, r.Fact)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved revocation: %+v\n", r)
}
