package schemas

import "github.com/spolu/settle/lib/db"

const (
	offersSQL = `
CREATE TABLE IF NOT EXISTS offers(
  owner VARCHAR(256) NOT NULL,       -- owner address
  token VARCHAR(256) NOT NULL,       -- token
  created TIMESTAMP NOT NULL,
  propagation VARCHAR(32) NOT NULL,  -- propagation type (canonical, propagated)

  base_asset VARCHAR(256) NOT NULL,  -- base asset name
  quote_asset VARCHAR(256) NOT NULL, -- quote asset name

  base_price VARCHAR(64) NOT NULL,   --  base asset price
  quote_price VARCHAR(64) NOT NULL,  -- quote asset price
  amount VARCHAR(64) NOT NULL,       -- amount of quote asset asked

  status VARCHAR(32) NOT NULL,       -- status (active, closed, consumed)
  remainder VARCHAR(64) NOT NULL,    -- remainder amount of quote asset asked

  PRIMARY KEY(owner, token)
);
`
)

func init() {
	db.RegisterSchema(
		"mint",
		"offers",
		offersSQL,
	)
}
