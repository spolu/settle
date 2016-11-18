package schemas

import "github.com/spolu/settle/mint/model"

const (
	operationsSQL = `
CREATE TABLE IF NOT EXISTS operations(
  user VARCHAR(256) NOT NULL,   -- user token
  owner VARCHAR(256) NOT NULL,  -- owner address
  token VARCHAR(256) NOT NULL,  -- token
  created TIMESTAMP NOT NULL,

  type VARCHAR(32) NOT NULL,    -- type (canonical, propagated)

  asset VARCHAR(256) NOT NULL,                     -- asset name
  source VARCHAR(512),                             -- source address
  destination VARCHAR(512),                        -- destination address
  amount VARCHAR(64) NOT NULL CHECK (amount > 0),  -- operation amount

  PRIMARY KEY(user, owner, token),
  CONSTRAINT operations_user_fk FOREIGN KEY (user) REFERENCES users(token)
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"operations",
		operationsSQL,
	)
}
