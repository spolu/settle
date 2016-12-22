package endpoint

import (
	"context"
	"regexp"

	"github.com/spolu/settle/lib/errors"
)

// Possible email: von.neumann+foo@ias.edu
var emailRegexp = regexp.MustCompile(
	"^([a-zA-Z0-9-_.]{1,256})(\\+[a-zA-Z0-9-_.]+){0,1}@" +
		"([a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+)$")

// Possible username: von.neuman-23_86
var usernameRegexp = regexp.MustCompile("^([a-zA-Z0-9-_.]{1,256})$")

// ValidateEmail vlaidates an email address.
func ValidateEmail(
	ctx context.Context,
	email string,
) (*string, error) {

	if !emailRegexp.MatchString(email) {
		return nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "email_invalid",
			"The email you provided is invalid: %s.",
			email,
		))
	}

	return &email, nil
}

// ValidateUsername vlaidates an email address.
func ValidateUsername(
	ctx context.Context,
	username string,
) (*string, error) {

	if !usernameRegexp.MatchString(username) {
		return nil, errors.Trace(errors.NewUserErrorf(nil,
			400, "username_invalid",
			"The username you provided is invalid: %s.",
			username,
		))
	}

	return &username, nil
}
