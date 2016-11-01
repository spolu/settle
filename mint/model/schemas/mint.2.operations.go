package schemas

import "github.com/spolu/settle/mint/model"

const (
	operationsSQL = `
CREATE TABLE IF NOT EXISTS operations(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL,
  livemode BOOL NOT NULL,

  asset VARCHAR(256) NOT NULL,                     -- asset token
  source VARCHAR(512),                             -- source user address
  destination VARCHAR(512),                        -- destination user address
  amount VARCHAR(64) NOT NULL CHECK (amount > 0),  -- operation amount

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
