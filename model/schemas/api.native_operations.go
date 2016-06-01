package model

const (
	nativeOperationsSQL = `
CREATE TABLE IF NOT EXISTS native_operations(
  id BIGSERIAL,
  token VARCHAR(256) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT UTC_NOW(),
  livemode BOOL NOT NULL,

  user_token VARCHAR(256) NOT NULL,
  user_update_id BIGINT NOT NULL,

  type VARCHAR(64) NOT NULL,
  parameters JSON NOT NULL,
  transaction_xdr TEXT NOT NULL

  PRIMARY KEY(id),
  UNIQUE(token)
);
CREATE INDEX native_operations_created_idx ON native_operations (created);
`
)

func init() {
	registerSchema(
		"api",
		"native_operations",
		nativeOperationsSQL,
	)
}
