package register

import (
	"context"
	"net/smtp"

	"github.com/spolu/settle/lib/env"
)

const (
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
	// EnvCfgReCAPTCHASecret is the env config key for the reCAPTCHA secret to
	// use to verify users.
	EnvCfgReCAPTCHASecret env.ConfigKey = "recaptcha_secret"
)

// GetMint retrieves the current mint host from the given contest.
func GetMint(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgMint]
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
	a := smtp.PlainAuth("", smtpLogin, smtpPassword, smtpHost)

	return &a, smtpHost
}

// GetReCAPTCHASecret retrieves the reCAPTCHA secret.
func GetReCAPTCHASecret(
	ctx context.Context,
) string {
	return env.Get(ctx).Config[EnvCfgReCAPTCHASecret]
}
