// OWNER: stan

package schemas

import "github.com/spolu/settle/mint/model"

const (
	updatesSQL = `
CREATE TABLE IF NOT EXISTS updates(
  token VARCHAR(256) NOT NULL,  -- token
  created TIMESTAMP NOT NULL,

  type VARCHAR(32) NOT NULL,         -- type (canonical, propagated)

  subject_owner VARCHAR(256) NOT NULL,
  subject_token VARCHAR(256) NOT NULL,

  source VARCHAR(256) NOT NULL,      -- source mint host
  destination VARCHAR(256) NOT NULL, -- destination mint host

  status VARCHAR(32) NOT NULL,       -- status (pending, succeeded, failed)
  attempts INT NOT NULL,             -- attempts count

  PRIMARY KEY(token)
);
`
)

func init() {
	model.RegisterSchema(
		"mint",
		"updates",
		updatesSQL,
	)
}
