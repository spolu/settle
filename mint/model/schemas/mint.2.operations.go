package schemas

import "github.com/spolu/settle/lib/db"

const (
	operationsSQL = `
CREATE TABLE IF NOT EXISTS operations(
  owner VARCHAR(256) NOT NULL,       -- owner address
  token VARCHAR(256) NOT NULL,       -- token
  created TIMESTAMP NOT NULL,
  propagation VARCHAR(32) NOT NULL,  -- propagation type (canonical, propagated)

  asset VARCHAR(256) NOT NULL,       -- asset name
  source VARCHAR(256) NOT NULL,      -- source address
  destination VARCHAR(256) NOT NULL, -- destination address
  amount VARCHAR(64) NOT NULL,       -- operation amount

  status VARCHAR(32) NOT NULL,       -- status (reserved, settled, canceled)
  txn VARCHAR(256),                  -- transaction id
  hop SMALLINT,                      -- transaction hop

  PRIMARY KEY(owner, token)
);
`
)

func init() {
	db.RegisterSchema(
		"mint",
		"operations",
		operationsSQL,
	)
}
