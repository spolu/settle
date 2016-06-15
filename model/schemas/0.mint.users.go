package model

const (
	usersSQL = `
CREATE TABLE IF NOT EXISTS users(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  username VARCHAR(256) NOT NULL,
  email VARCHAR(256) NOT NULL,
  scrypt VARCHAR(256) NOT NULL,

  PRIMARY KEY(token),
  UNIQUE(username)
);
`
)

func init() {
	registerSchema(
		"mint",
		"users",
		usersSQL,
	)
}
