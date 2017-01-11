package command

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
)

const (
	// CmdNmLogin is the command name.
	CmdNmLogin cli.CmdName = "login"
)

func init() {
	cli.Registrar[CmdNmLogin] = NewLogin
}

// Login a user up to a certain amount of a given asset they issued.
type Login struct {
}

// NewLogin constructs and initializes the command.
func NewLogin() cli.Command {
	return &Login{}
}

// Name returns the command name.
func (c *Login) Name() cli.CmdName {
	return CmdNmLogin
}

// Help prints out the help message for the command.
func (c *Login) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle login\n")
	out.Normf("\n")
	out.Normf("  Logging in will store your credentials locally under:\n")
	out.Valuf("  ~/.settle/credentials-" + string(env.Get(ctx).Environment) + "\n")
	out.Normf("\n")
	out.Normf("  The credentials from your mint are composed of your user address (of the form \n  ")
	out.Valuf("von.neumann@ias.edu")
	out.Normf(" where ")
	out.Valuf("ias.edu")
	out.Normf(" is your mint) along with your password.\n")
	out.Normf("\n")
	out.Normf("  If you don't already have access to a mint, you can register on publicly\n")
	out.Normf("  accessible mints using: ")
	out.Boldf("settle register")
	out.Normf("\n\n")
}

// Parse parses the arguments passed to the command.
func (c *Login) Parse(
	ctx context.Context,
	args []string,
) error {
	return nil
}

// Execute the command or return a human-friendly error.
func (c *Login) Execute(
	ctx context.Context,
) error {

	reader := bufio.NewReader(os.Stdin)

	out.Normf("    Address []: ")
	address, _ := reader.ReadString('\n')

	out.Normf("    Password []: ")
	password, _ := reader.ReadString('\n')

	out.Normf("Is the information correct? [Y/n]: ")
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(confirmation)
	if confirmation != "" && confirmation != "Y" {
		return errors.Trace(errors.Newf("Registration aborted by user."))
	}

	err := cli.Login(ctx,
		strings.TrimSpace(address), strings.TrimSpace(password))
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
