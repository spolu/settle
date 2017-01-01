// OWNER: stan

package schemas

import "github.com/spolu/settle/lib/db"

const (
	usersSQL = `
CREATE TABLE IF NOT EXISTS users(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL,

  status VARCHAR(32) NOT NULL,    -- status (unverified, verified)
  username VARCHAR(256) NOT NULL,
  email VARCHAR(256) NOT NULL,

  secret VARCHAR(256) NOT NULL,   -- secret sent over email
  password VARCHAR(256) NOT NULL, -- rollable password

  mint_token VARCHAR(256),        -- the mint user token

  PRIMARY KEY(token),
  CONSTRAINT users_username_u UNIQUE (username)
);
`
)

func init() {
	db.RegisterSchema(
		"register",
		"users",
		usersSQL,
	)
}
