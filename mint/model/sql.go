// OWNER: stan

package model

import (
	"context"
	"fmt"
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

// NewSqlite3DBForPath returns a new sqlite3 DB stored at the provided path or
// defaulting to `~/.mint/mint-$env.dr`.
func NewSqlite3DBForPath(
	ctx context.Context,
	path string,
) (*sqlx.DB, error) {
	err := error(nil)

	if path == "" {
		path, err = homedir.Expand(
			fmt.Sprintf("~/.mint/mint-%s.db", env.Get(ctx).Environment))
		if err != nil {
			log.Fatal(errors.Details(err))
		}
	}
	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return nil, err
	}

	mintDB, err := sqlx.Connect("sqlite3", path)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Opened sqlite3 DB: path=%s\n", path)

	return mintDB, nil
}
