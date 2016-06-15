package schemas

import "github.com/spolu/settle/model"

const (
	balancesSQL = `
CREATE TABLE IF NOT EXISTS balances(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  asset VARCHAR(256) NOT NULL,  -- the balance's asset token
  owner VARCHAR(256) NOT NULL,  -- the balance's owner's user token
  value NUMERIC(39) NOT NULL,   -- the balance's value

  PRIMARY KEY(token),
  CONSTRAINT balances_asset_fk FOREIGN KEY (asset) REFERENCES assets(token),
  CONSTRAINT balances_owner_fk FOREIGN KEY (owner) REFERENCES users(token)
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
