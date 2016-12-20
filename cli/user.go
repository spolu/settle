// OWNER stan

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/mint"
)

// Credentials rerpesents the credentials of the currently logged in user.
type Credentials struct {
	Username string `json:"username"`
	Host     string `json:"mint"`
	Password string `json:"password"`
}

const (
	// credentialsKey the context.Context key to store the credentials
	credentialsKey ContextKey = "cli.credentials"
)

// WithCredentials stores the credentials in the provided context.
func WithCredentials(
	ctx context.Context,
	credentials *Credentials,
) context.Context {
	return context.WithValue(ctx, credentialsKey, credentials)
}

// GetCredentials returns the credentials currently stored in the context.
func GetCredentials(
	ctx context.Context,
) *Credentials {
	return ctx.Value(credentialsKey).(*Credentials)
}

// CredentialsPath returns the crendentials path for the current environment.
func CredentialsPath(
	ctx context.Context,
) (*string, error) {
	path, err := homedir.Expand(
		fmt.Sprintf("~/.settle/credentials-%s.json", env.Get(ctx).Environment))
	if err != nil {
		return nil, errors.Trace(err)
	}

	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &path, nil
}

// CurrentUser retrieves the current user by reading CredentialsPath.
func CurrentUser(
	ctx context.Context,
) (*Credentials, error) {
	path, err := CredentialsPath(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if _, err := os.Stat(*path); os.IsNotExist(err) {
		return nil, nil
	}

	raw, err := ioutil.ReadFile(*path)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var c Credentials
	err = json.Unmarshal(raw, &c)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &c, nil
}

// Login logs the user in by storing its credentials after valdation in
// CredentialsPath.
func Login(
	ctx context.Context,
	address string,
	password string,
) error {
	username, host, err := mint.UsernameAndMintHostFromAddress(ctx, address)
	if err != nil {
		return errors.Trace(err)
	}
	creds := &Credentials{
		Host:     host,
		Username: username,
		Password: password,
	}

	// TOOD(stan): check credentials

	path, err := CredentialsPath(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	formatted, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return errors.Trace(err)
	}

	err = ioutil.WriteFile(*path, formatted, 0644)
	if err != nil {
		return errors.Trace(err)
	}

	out.Statf("[Storing credentials] file=%s\n", *path)

	return nil
}

// Logout logs the user out by destoying its credentials at CredentialsPath.
func Logout(
	ctx context.Context,
) error {
	path, err := CredentialsPath(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	err = os.Remove(*path)
	if err != nil {
		return errors.Trace(err)
	}

	out.Statf("[Erasing credentials] file=%s\n", *path)

	return nil
}
