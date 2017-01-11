package schemas

import "github.com/spolu/settle/lib/db"

const (
	usersSQL = `
CREATE TABLE IF NOT EXISTS users(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL,

  username VARCHAR(256) NOT NULL,
  password_hash VARCHAR(256) NOT NULL,

  PRIMARY KEY(token),
  CONSTRAINT users_username_u UNIQUE (username)
);
`
)

func init() {
	db.RegisterSchema(
		"mint",
		"users",
		usersSQL,
	)
}
