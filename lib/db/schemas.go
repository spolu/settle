package db

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
)

var schemas = map[string]map[string]string{}

// RegisterSchema lets schemas register themselves.
func RegisterSchema(
	tag string,
	table string,
	schema string,
) {
	if _, ok := schemas[tag]; !ok {
		schemas[tag] = map[string]string{}
	}
	schemas[tag][table] = schema
}

// CreateDBTables creates the Mint DB tables if they don't exist.
func CreateDBTables(
	ctx context.Context,
	tag string,
	db *sqlx.DB,
) error {
	for name, sch := range schemas[tag] {
		logging.Logf(ctx, "Executing schema: tag=%s name=%s\n", tag, name)
		_, err := db.Exec(sch)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
