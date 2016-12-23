package register

import (
	"context"
	"net/smtp"
	"strings"

	"github.com/spolu/settle/lib/env"
)

const (
	// EnvCfgHost is the env config key for the register host.
	EnvCfgHost env.ConfigKey = "host"
	// EnvCfgPort is the port on which to run the register.
	EnvCfgPort env.ConfigKey = "port"
	// EnvCfgKeyFile is the production certificate key file.
	EnvCfgKeyFile env.ConfigKey = "key_file"
	// EnvCfgCrtFile is the production certificate file.
	EnvCfgCrtFile env.ConfigKey = "crt_file"
	// EnvCfgCredsURL is the URL that is sent to the user over email to
	// retrieve their credentials.
	EnvCfgCredsURL env.ConfigKey = "credentials_url"
	// EnvCfgMint is the env config key for the mint this register service is
	// bound to.
	EnvCfgMint env.ConfigKey = "mint"
	// EnvCfgSMTPLogin is the env config key for the SMTP login to use to send
	// verification emails.
	EnvCfgSMTPLogin env.ConfigKey = "smtp_login"
	// EnvCfgSMTPPassword is the env config key for the SMTP password to use to
	// send verification emails.
	EnvCfgSMTPPassword env.ConfigKey = "smtp_password"
	// EnvCfgSMTPHost is the env config key for the SMTP host to use to send
	// verification emails.
	EnvCfgSMTPHost env.ConfigKey = "smtp_host"
	// EnvCfgFrom is the email address to send registration emails from.
	EnvCfgFrom env.ConfigKey = "from"
	// EnvCfgReCAPTCHASecret is the env config key for the reCAPTCHA secret to
	// use to verify users.
	EnvCfgReCAPTCHASecret env.ConfigKey = "recaptcha_secret"
)

// GetHost retrieves the current register host from the given contest.
func GetHost(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgHost]
}

// GetPort retrieves the current register port from the given contest.
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

// GetCrtFile retrieves the production certificate key file.
func GetCrtFile(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgCrtFile]
}

// GetMint retrieves the current mint host from the given contest.
func GetMint(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgMint]
}

// GetCredsURL retrieves the credentials URL for users to retrieve their
// credentials.
func GetCredsURL(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgCredsURL]
}

// GetSMTP retrieves the SMTP credentials.
func GetSMTP(
	ctx context.Context,
) (*smtp.Auth, string) {

	smtpLogin := env.Get(ctx).Config[EnvCfgSMTPLogin]
	smtpPassword := env.Get(ctx).Config[EnvCfgSMTPPassword]
	smtpHost := env.Get(ctx).Config[EnvCfgSMTPHost]

	if smtpLogin == "" || smtpHost == "" {
		return nil, smtpHost
	}
	a := smtp.PlainAuth("",
		smtpLogin, smtpPassword, strings.Split(smtpHost, ":")[0])

	return &a, smtpHost
}

// GetFrom retrieves the current address to send registration emails from
func GetFrom(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgFrom]
}

// GetReCAPTCHASecret retrieves the reCAPTCHA secret.
func GetReCAPTCHASecret(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgReCAPTCHASecret]
}
