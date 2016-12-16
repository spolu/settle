// OWNER stan

package command

import (
	"context"

	"github.com/spolu/settle/cli"
)

const (
	// CmdNmTrust is the command name.
	CmdNmTrust cli.CmdName = "trust"
)

func init() {
	cli.Registrar[CmdNmTrust] = NewTrust
}

// Trust a user up to a certain amount of a given asset they issued.
type Trust struct {
}

// NewTrust constructs and initializes the command.
func NewTrust() cli.Command {
	return &Trust{}
}

// Name returns the command name.
func (c *Trust) Name() cli.CmdName {
	return CmdNmTrust
}

// Help prints out the help message for the command.
func (c *Trust) Help(
	ctx context.Context,
) {
}

// Parse parses the arguments passed to the command.
func (c *Trust) Parse(
	ctx context.Context,
	args []string,
) error {
	return nil
}

// Execute the command or return a human-friendly error.
func (c *Trust) Execute(
	ctx context.Context,
) error {
	return nil
}
