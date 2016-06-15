package schemas

import "github.com/spolu/settle/model"

const (
	operationsSQL = `
CREATE TABLE IF NOT EXISTS operations(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  asset VARCHAR(256) NOT NULL,       -- the opeation's asset token
  source VARCHAR(256) NOT NULL,      -- the operation's source user token
  destination VARCHAR(256) NOT NULL, -- the operation's destination user token
  amount NUMERIC(39) NOT NULL,       -- the operation's amount

  PRIMARY KEY(token),
  CONSTRAINT operations_asset_fk FOREIGN KEY (asset) REFERENCES assets(token),
  CONSTRAINT operations_source_fk FOREIGN KEY (source) REFERENCES users(token),
  CONSTRAINT operations_destination_fk FOREIGN KEY (destination) REFERENCES users(token)
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
