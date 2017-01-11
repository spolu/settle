package schemas

import "github.com/spolu/settle/lib/db"

const (
	tasksSQL = `
CREATE TABLE IF NOT EXISTS tasks(
  token VARCHAR(256) NOT NULL,  -- token
  created TIMESTAMP NOT NULL,

  name VARCHAR(256) NOT NULL,                      -- task name
  subject VARCHAR(256) NOT NULL,                   -- task subject

  status VARCHAR(32) NOT NULL,       -- status (pending, succeeded, failed)
  retry INT,                         -- retry count

  PRIMARY KEY(token)
);
`
)

func init() {
	db.RegisterSchema(
		"mint",
		"tasks",
		tasksSQL,
	)
}
