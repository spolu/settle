package schemas

import "github.com/spolu/settle/lib/db"

const (
	crossingsSQL = `
CREATE TABLE IF NOT EXISTS crossings(
  owner VARCHAR(256) NOT NULL,       -- owner address
  token VARCHAR(256) NOT NULL,       -- token
  created TIMESTAMP NOT NULL,
  propagation VARCHAR(32) NOT NULL,  -- propagation type (canonical, propagated)

  offer VARCHAR(256) NOT NULL,  -- offer id
  amount VARCHAR(64) NOT NULL,  -- crossing amount

  status VARCHAR(32) NOT NULL,  -- status (reserved, settled, canceled)
  txn VARCHAR(256) NOT NULL,    -- transaction id
  hop SMALLINT NOT NULL,        -- transaction hop

  PRIMARY KEY(owner, token)
);
`
)

func init() {
	db.RegisterSchema(
		"mint",
		"crossings",
		crossingsSQL,
	)
}
