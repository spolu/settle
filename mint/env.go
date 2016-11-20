package mint

import (
	"context"

	"github.com/spolu/settle/lib/env"
)

const (
	// EnvCfgMintHost is the env config key for the mint host.
	EnvCfgMintHost env.ConfigKey = "mint_host"
)

// GetMintHost retrieves the current mint host from the given contest.
func GetMintHost(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgMintHost]
}
