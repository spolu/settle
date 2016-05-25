package model

const (
	userUpdatesSQL = `
CREATE TABLE IF NOT EXISTS user_updates(
  id BIGSERIAL,
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  creation TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  username VARCHAR(256),
  address VARCHAR(256),
  encrypted_seed VARCHAR(256),

  email VARCHAR(256),
  verifier VARCHAR(256),

  PRIMARY KEY(id),
  UNIQUE(token),
  UNIQUE(username),
  UNIQUE(address),
  UNIQUE(email)
);
`
)

func init() {
	registerSchema(
		"api",
		"user_updates",
		userUpdatesSQL,
	)
}
