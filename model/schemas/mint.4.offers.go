package schemas

import "github.com/spolu/settle/model"

const (
	offersSQL = `
CREATE TYPE OFFERTYPE AS ENUM ('bid', 'ask');
CREATE TYPE OFFERSTATUS AS ENUM ('active', 'closed');
CREATE TABLE IF NOT EXISTS offers(
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  owner VARCHAR(256) NOT NULL,       -- the offer's owner's address
  base_asset VARCHAR(256) NOT NULL,  -- the base asset
  quote_asset VARCHAR(256) NOT NULL, -- the quote asset

  type OFFERTYPE NOT NULL,          -- the type (bid, ask)

  base_price NUMERIC(39) NOT NULL,  -- the base asset price
  quote_price NUMERIC(39) NOT NULL, -- the quote asset price
  amount NUMERIC(39) NOT NULL,  -- the amount of quote asset offered

  status OFFERSTATUS NOT NULL,  -- the status (active, closed)

  PRIMARY KEY(token)
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"offers",
		offersSQL,
	)
}
