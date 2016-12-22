package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
)

// NewDBForDSN parses the provided DSN and returns the initialized DB for it.
// Supported DSN are:
// ```
//  sqlite3:///home/spolu/foo.db
//  sqlite3://:memory:
//  postgres://foo:password@localhost/mydb?sslmode=verify-full
// ```
// If no DSN is specified, the default DSN is used instead.
func NewDBForDSN(
	ctx context.Context,
	dsn string,
	defaultDSN string,
) (*sqlx.DB, error) {
	if dsn == "" {
		dsn = defaultDSN
	}

	c := strings.Split(dsn, "://")
	if len(c) != 2 {
		return nil, errors.Trace(errors.Newf("Invalid DB DSN: %s", dsn))
	}
	switch c[0] {
	case "sqlite3":
		switch c[1] {
		case ":memory:":
			return NewSqlite3DBInMemory(ctx)
		default:
			return NewSqlite3DBForPath(ctx, c[1])
		}
	case "postgres":
		return NewPostgresDBForDSN(ctx, dsn)
	default:
		return nil, errors.Trace(errors.Newf("Non supported DB DSN: %s", dsn))
	}
}

// NewPostgresDBForDSN returns a new Postgres DB found at the DSN provided.
func NewPostgresDBForDSN(
	ctx context.Context,
	dsn string,
) (*sqlx.DB, error) {
	mintDB, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	logging.Logf(ctx, "Opened postgres DB: dsn=%s\n", dsn)

	return mintDB, nil
}

// NewSqlite3DBForPath returns a new sqlite3 DB stored at the provided path or
// defaulting to `~/.mint/mint-$env.dr`.
func NewSqlite3DBForPath(
	ctx context.Context,
	path string,
) (*sqlx.DB, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, errors.Trace(err)
	}

	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return nil, errors.Trace(err)
	}

	mintDB, err := sqlx.Connect("sqlite3", path)
	if err != nil {
		return nil, errors.Trace(err)
	}
	logging.Logf(ctx, "Opened sqlite3 DB: in_memory=false path=%s\n", path)

	return mintDB, nil
}

// NewSqlite3DBInMemory returns a new in-memory sqlite3 DB.
func NewSqlite3DBInMemory(
	ctx context.Context,
) (*sqlx.DB, error) {
	err := error(nil)

	mintDB, err := sqlx.Connect("sqlite3", ":memory:?_txlock=exclusive")
	if err != nil {
		return nil, errors.Trace(err)
	}
	logging.Logf(ctx, "Opened sqlite3 DB: in_memory=true\n")

	return mintDB, nil
}
