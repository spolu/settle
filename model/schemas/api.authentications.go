package model

const (
	authenticationsSQL = `
CREATE TABLE IF NOT EXISTS authentications(
  id BIGSERIAL,
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  method VARCHAR(8) NOT NULL,
  url VARCHAR(4096) NOT NULL,

  challenge VARCHAR(256),
  address VARCHAR(256),
  signature VARCHAR(256),

  PRIMARY KEY(id),
  UNIQUE(token),
  UNIQUE(challenge)
);
`
)

func init() {
	registerSchema(
		"api",
		"authentications",
		authenticationsSQL,
	)
}
