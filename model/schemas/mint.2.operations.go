package schemas

import "github.com/spolu/settle/model"

const (
	operationsSQL = `
CREATE TABLE IF NOT EXISTS operations(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  asset VARCHAR(256) NOT NULL,       -- the opeation's asset token
  source VARCHAR(512),               -- the operation's source user address
  destination VARCHAR(512) NOT NULL, -- the operation's destination user address
  amount NUMERIC(39) NOT NULL,       -- the operation's amount

  PRIMARY KEY(token),
  CONSTRAINT operations_asset_fk FOREIGN KEY (asset) REFERENCES assets(token)
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"operations",
		operationsSQL,
	)
}
