package schemas

import "github.com/spolu/settle/mint/model"

const (
	assetsSQL = `
CREATE TABLE IF NOT EXISTS assets(
  user VARCHAR(256) NOT NULL,   -- user token
  owner VARCHAR(256) NOT NULL,  -- owner address
  token VARCHAR(256) NOT NULL,  -- token
  created TIMESTAMP NOT NULL,

  code VARCHAR(64) NOT NULL,    -- the code of the asset
  scale SMALLINT,               -- factor by which the asset native is scaled

  PRIMARY KEY(user, owner, token),
  CONSTRAINT assets_user_fk FOREIGN KEY (user) REFERENCES users(token),
  CONSTRAINT assets_owner_code_u UNIQUE (owner, code) -- not propagated
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"assets",
		assetsSQL,
	)
}
