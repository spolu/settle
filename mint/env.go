package mint

import (
	"context"
	"fmt"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/logging"
)

const (
	// EnvCfgHost is the env config key for the mint host.
	EnvCfgHost env.ConfigKey = "host"
	// EnvCfgPort is the port on which to run the mint.
	EnvCfgPort env.ConfigKey = "port"
	// EnvCfgKeyFile is the production certificate key file.
	EnvCfgKeyFile env.ConfigKey = "key_file"
	// EnvCfgCertFile is the production certificate file.
	EnvCfgCertFile env.ConfigKey = "cert_file"
)

// GetHost retrieves the current mint host from the given contest.
func GetHost(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgHost]
}

// GetPort retrieves the current mint port from the given contest.
func GetPort(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgPort]
}

// GetKeyFile retrieves the production certificate key file.
func GetKeyFile(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgKeyFile]
}

// GetCertFile retrieves the production certificate key file.
func GetCertFile(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgCertFile]
}

// Logf shells out to logging.Logf adding the mint host as prefix.
func Logf(
	ctx context.Context,
	format string,
	v ...interface{},
) {
	logging.Logf(ctx, fmt.Sprintf("[%s] ", GetHost(ctx))+format, v...)
}
