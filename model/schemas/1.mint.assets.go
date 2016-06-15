package model

const (
	assetsSQL = `
CREATE TABLE IF NOT EXISTS assets(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  issuer VARCHAR(256) NOT NULL, -- the asset's issuer's user token
  code VARCHAR(64) NOT NULL,    -- the name of the asset
  scale SMALLINT,               -- factor by which the asset native is scaled

  PRIMARY KEY(token),
  CONSTRAINT assets_issuer_code_u UNIQUE (issuer, code),
  CONSTRAINT assets_issuer_fk FOREIGN KEY (issuer) REFERENCES users(token)
);
`
)

func init() {
	registerSchema(
		"mint",
		"assets",
		assetsSQL,
	)
}
