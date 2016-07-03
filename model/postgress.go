package model

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/peer-currencies/lib/errors"

	// pq is used as underlying sql driver.
	_ "github.com/lib/pq"
)

var mintDB *sqlx.DB

func ensureMintDB() {
	if mintDB != nil {
		return
	}

	err := error(nil)
	mintDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("MINT_DB_HOST"),
		os.Getenv("MINT_DB_PORT"),
		os.Getenv("MINT_DB_USER"),
		os.Getenv("MINT_DB_PASSWORD"),
		os.Getenv("MINT_DB_NAME"),
	)
	mintDB, err = sqlx.Connect("postgres", mintDSN)
	if err != nil {
		log.Fatal(errors.Details(err))
	} else {
		fmt.Printf("Initialized mintDB with DSN: %s\n", mintDSN)
	}
}

func init() {
	ensureMintDB()
}

// MintDB returns the mintDB singeleton.
func MintDB() *sqlx.DB {
	return mintDB
}

// Shutdown attempts to close all existing DB connections.
func Shutdown() {
	if mintDB != nil {
		mintDB.Close()
	}
}

// MustClose is used to ensure statement get closed.
func MustClose(statement io.Closer) {
	if err := statement.Close(); err != nil {
		panic(err)
	}
}
