// OWNER: stan

package schemas

import "github.com/spolu/settle/mint/model"

const (
	transactionsSQL = `
CREATE TABLE IF NOT EXISTS transactions(
  user VARCHAR(256) NOT NULL,   -- user token
  owner VARCHAR(256) NOT NULL,  -- owner address
  token VARCHAR(256) NOT NULL,  -- token
  created TIMESTAMP NOT NULL,

  propagation VARCHAR(32) NOT NULL,  -- propagation type (canonical, propagated)

  base_asset VARCHAR(256) NOT NULL,  -- base asset name
  quote_asset VARCHAR(256) NOT NULL, -- quote asset name

  base_price VARCHAR(64) NOT NULL,   --  base asset price
  quote_price VARCHAR(64) NOT NULL,  -- quote asset price
  amount VARCHAR(64) NOT NULL,       -- amount of quote asset asked

  path VARCHAR(2048) NOT NULL,       -- join of offer ids

  PRIMARY KEY(user, owner, token),
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
