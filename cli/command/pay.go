// OWNER stan

package command

import (
	"context"

	"github.com/spolu/settle/cli"
)

const (
	// CmdNmPay is the command name.
	CmdNmPay cli.CmdName = "pay"
)

func init() {
	cli.Registrar[CmdNmPay] = NewPay
}

// Pay a user up to a certain amount of a given asset they issued.
type Pay struct {
}

// NewPay constructs and initializes the command.
func NewPay() cli.Command {
	return &Pay{}
}

// Name returns the command name.
func (c *Pay) Name() cli.CmdName {
	return CmdNmPay
}

// Help prints out the help message for the command.
func (c *Pay) Help(
	ctx context.Context,
) {
}

// Parse parses the arguments passed to the command.
func (c *Pay) Parse(
	ctx context.Context,
	args []string,
) error {
	return nil
}

// Execute the command or return a human-friendly error.
func (c *Pay) Execute(
	ctx context.Context,
) error {
	return nil
}
