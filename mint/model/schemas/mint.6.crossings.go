// OWNER: stan

package schemas

import "github.com/spolu/settle/mint/model"

const (
	crossingsSQL = `
CREATE TABLE IF NOT EXISTS crossings(
  user VARCHAR(256) NOT NULL,   -- user token
  owner VARCHAR(256) NOT NULL,  -- owner address
  token VARCHAR(256) NOT NULL,  -- token
  created TIMESTAMP NOT NULL,

  offer VARCHAR(256) NOT NULL,                     -- offer id
  amount VARCHAR(64) NOT NULL CHECK (amount > 0),  -- crossing amount

  status VARCHAR(32) NOT NULL,       -- status (reserved, settled, canceled)
  txn VARCHAR(256) NOT NULL,         -- transaction id

  PRIMARY KEY(user, owner, token),
  CONSTRAINT crossings_user_fk FOREIGN KEY (user) REFERENCES users(token)
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"crossings",
		crossingsSQL,
	)
}
