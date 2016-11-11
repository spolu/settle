// OWNER: stan

package model

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/logging"
)

var schemas = map[string]map[string]string{
	"mint": map[string]string{},
}

// RegisterSchema lets schemas register themselves.
func RegisterSchema(
	db string,
	table string,
	schema string,
) {
	schemas[db][table] = schema
}

// CreateMintDBTables creates the Mint DB tables if they don't exist.
func CreateMintDBTables(
	ctx context.Context,
	db *sqlx.DB,
) error {
	for name, sch := range schemas["mint"] {
		logging.Logf(ctx, "Executing schema: %s\n", name)
		_, err := db.Exec(sch)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
