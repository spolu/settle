package model

const (
	userUpdatesSQL = `
CREATE TABLE IF NOT EXISTS user_updates(
  id BIGSERIAL,
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  is_active BOOL NOT NULL,
  user_token VARCHAR(256) NOT NULL,
  creation TIMESTAMP NOT NULL,
  username VARCHAR(256) NOT NULL,
  address VARCHAR(256) NOT NULL,
  encrypted_seed VARCHAR(256) NOT NULL,

  email VARCHAR(256) NOT NULL,
  verifier VARCHAR(256) NOT NULL,

  PRIMARY KEY(id),
  UNIQUE(token)
);
CREATE INDEX user_updates_user_token_is_active_idx ON user_updates (user_token, is_active);
CREATE INDEX user_updates_username_is_active_idx ON user_updates (username, is_active);
CREATE INDEX user_updates_address_is_active_idx ON user_updates (address, is_active);
`
)

func init() {
	registerSchema(
		"api",
		"user_updates",
		userUpdatesSQL,
	)
}
