package model

import "github.com/spolu/settle/lib/errors"

var schemas = map[string]map[string]string{
	"api": map[string]string{},
}

func registerSchema(
	db string,
	table string,
	schema string,
) {
	schemas[db][table] = schema
}

const (
	toolsSQL = `
CREATE OR REPLACE FUNCTION utc_now() RETURNS TIMESTAMP AS $$
  SELECT CLOCK_TIMESTAMP() AT TIME ZONE 'utc'
$$ language sql;
`
)

func init() {
	registerSchema(
		"api",
		"_tools",
		toolsSQL,
	)
}

// CreateAPIDBTables creates the API DB tables if they don't exist.
func CreateAPIDBTables() error {
	for t, sch := range schemas["api"] {
		_, err := apidb.Exec(sch)
		if err != nil {
			return errors.Trace(err)
		}
	}
}
