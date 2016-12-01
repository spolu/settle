package mint

import (
	"context"
	"fmt"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/logging"
)

const (
	// EnvCfgMintHost is the env config key for the mint host.
	EnvCfgMintHost env.ConfigKey = "mint_host"
)

// GetHost retrieves the current mint host from the given contest.
func GetHost(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgMintHost]
}

// Logf shells out to logging.Logf adding the mint host as prefix.
func Logf(
	ctx context.Context,
	format string,
	v ...interface{},
) {
	logging.Logf(ctx, fmt.Sprintf("[%s] ", GetHost(ctx))+format, v...)
}
