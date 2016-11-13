package schemas

import "github.com/spolu/settle/mint/model"

const (
	balancesSQL = `
CREATE TABLE IF NOT EXISTS balances(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL,

  asset VARCHAR(256) NOT NULL,  -- the balance's asset token
  owner VARCHAR(256) NOT NULL,  -- the balance's owner's address
  value VARCHAR(64) NOT NULL,   -- the balance's value

  PRIMARY KEY(token),
  CONSTRAINT balances_asset_fk FOREIGN KEY (asset) REFERENCES assets(token),
  CONSTRAINT balances_asset_owner_u UNIQUE (asset, owner)
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"balances",
		balancesSQL,
	)
}
