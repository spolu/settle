package schemas

import "github.com/spolu/settle/lib/db"

const (
	assetsSQL = `
CREATE TABLE IF NOT EXISTS assets(
  owner VARCHAR(256) NOT NULL,       -- owner address
  token VARCHAR(256) NOT NULL,       -- token
  created TIMESTAMP NOT NULL,
  propagation VARCHAR(32) NOT NULL,  -- propagation type (canonical, propagated)

  code VARCHAR(64) NOT NULL,    -- the code of the asset
  scale SMALLINT,               -- factor by which the asset native is scaled

  PRIMARY KEY(owner, token),
  CONSTRAINT assets_owner_code_scale_u UNIQUE (owner, code, scale)
);
`
)

func init() {
	db.RegisterSchema(
		"mint",
		"assets",
		assetsSQL,
	)
}
