package env

import "context"

// Environment is the type of the possible environment values.
type Environment string

// ConfigKey is the type of a config entry key.
type ConfigKey string

const (
	// Production is the production environment.
	Production Environment = "prod"
	// QA is the qa environment.
	QA Environment = "qa"
)

// Env is the value stored in context that stores the current environment along
// with configuration values.
type Env struct {
	Environment Environment
	Config      map[ConfigKey]string
}

// ContextKey is the type of the key used with context to carry contextual
// environment.
type ContextKey string

const (
	// envKey the context.Context key to store the env.
	envKey ContextKey = "env.env"
)

// With stores the environment in the provided context.
func With(
	ctx context.Context,
	env *Env,
) context.Context {
	return context.WithValue(ctx, envKey, env)
}

// Get returns the env currently stored in the context.
func Get(
	ctx context.Context,
) *Env {
	return ctx.Value(envKey).(*Env)
}
