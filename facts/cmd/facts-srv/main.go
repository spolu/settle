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

	s := model.NewSignature(
		f.ID,
		model.PublicKey("TESTENTITY"),
		"TESTSIGNATURE",
	)
	err = s.Save()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved signature: %+v\n", s)

	s, err = model.LoadSignature(s.ID, f.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved signature: %+v\n", s)
}
