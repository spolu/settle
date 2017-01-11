package command

import (
	"context"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
)

const (
	// CmdNmLogout is the command name.
	CmdNmLogout cli.CmdName = "logout"
)

func init() {
	cli.Registrar[CmdNmLogout] = NewLogout
}

// Logout a user up to a certain amount of a given asset they issued.
type Logout struct {
}

// NewLogout constructs and initializes the command.
func NewLogout() cli.Command {
	return &Logout{}
}

// Name returns the command name.
func (c *Logout) Name() cli.CmdName {
	return CmdNmLogout
}

// Help prints out the help message for the command.
func (c *Logout) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle logout\n")
	out.Normf("\n")
	out.Normf("  Logging out erases all locally stored credentials.\n")
	out.Normf("\n")
}

// Parse parses the arguments passed to the command.
func (c *Logout) Parse(
	ctx context.Context,
	args []string,
) error {
	return nil
}

// Execute the command or return a human-friendly error.
func (c *Logout) Execute(
	ctx context.Context,
) error {
	err := cli.Logout(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
