package model

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/mitchellh/go-homedir"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"

	// sqlite is used as underlying driver
	_ "github.com/mattn/go-sqlite3"
)

var mintDB *sqlx.DB

func ensureMintDB() {
	if mintDB != nil {
		return
	}
	err := error(nil)

	path := os.Getenv("MINT_DB_PATH")
	if path == "" {
		path, err = homedir.Expand(
			fmt.Sprintf("~/.mint/mint-%s.db", env.Current))
		if err != nil {
			log.Fatal(errors.Details(err))
		}
	}
	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		log.Fatal(errors.Details(err))
	}

	mintDB, err = sqlx.Connect("sqlite3", path)
	if err != nil {
		log.Fatal(errors.Details(err))
	} else {
		fmt.Printf("Opened sqlite3 mintDB: path=%s\n", path)
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
