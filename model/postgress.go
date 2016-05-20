package model

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settl/util/errors"

	// pq is used as underlying sql driver.
	_ "github.com/lib/pq"
)

var apidb *sqlx.DB

func init() {
	err := error(nil)
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("API_DB_HOST"),
		os.Getenv("API_DB_PORT"),
		os.Getenv("API_DB_USER"),
		os.Getenv("API_DB_PASSWORD"),
		os.Getenv("API_DB_NAME"),
	)
	apidb, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatal(errors.Details(err))
	} else {
		fmt.Printf("Initialized apidb with dsn: %s\n", dsn)
	}
}

// Shutdown attempts to close all existing DB connections.
func Shutdown() {
	if apidb != nil {
		apidb.Close()
	}
}
