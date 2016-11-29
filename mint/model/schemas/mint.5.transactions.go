// OWNER: stan

package schemas

import "github.com/spolu/settle/mint/model"

const (
	transactionsSQL = `
CREATE TABLE IF NOT EXISTS transactions(
  user VARCHAR(256),            -- user token (not present if propagated)
  owner VARCHAR(256) NOT NULL,  -- owner address
  token VARCHAR(256) NOT NULL,  -- token
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

  PRIMARY KEY(owner, token),
  CONSTRAINT transactions_user_fk FOREIGN KEY (user) REFERENCES users(token)
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"transactions",
		transactionsSQL,
	)
}
