package model

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settl/util/errors"

	// pq is used as underlying sql driver
	_ "github.com/lib/pq"
)

var apidb *sqlx.DB

func init() {
	apidb, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("API_DB_HOST"),
			os.Getenv("API_DB_PORT"),
			os.Getenv("API_DB_USER"),
			os.Getenv("API_DB_PASSWORD"),
			os.Getenv("API_DB_NAME"),
		))
	if err != nil {
		log.Fatal(errors.Details(err))
	} else {
		fmt.Println("Initialized apidb")
	}
}

// Shutdown attempts to close all existing DB connections.
func Shutdown() {
	if apidb != nil {
		apidb.Close()
	}
}
