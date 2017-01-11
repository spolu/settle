package schemas

import "github.com/spolu/settle/lib/db"

const (
	transactionsSQL = `
CREATE TABLE IF NOT EXISTS transactions(
  owner VARCHAR(256) NOT NULL,       -- owner address
  token VARCHAR(256) NOT NULL,       -- token
  created TIMESTAMP NOT NULL,
  propagation VARCHAR(32) NOT NULL,  -- propagation type (canonical, propagated)

  base_asset VARCHAR(256) NOT NULL,  -- base asset name
  quote_asset VARCHAR(256) NOT NULL, -- quote asset name
  amount VARCHAR(64) NOT NULL,       -- amount of quote asset asked
  destination VARCHAR(256) NOT NULL, -- the recipient address
  path VARCHAR(2048) NOT NULL,       -- join of offer ids

  status VARCHAR(32) NOT NULL,       -- status (reserved, settled, canceled)
  lock VARCHAR(256) NOT NULL,        -- lock = hex(scrypt(secret, id))
  secret VARCHAR(256),               -- lock secret

  PRIMARY KEY(owner, token)
);
`
)

func init() {
	db.RegisterSchema(
		"mint",
		"transactions",
		transactionsSQL,
	)
}
