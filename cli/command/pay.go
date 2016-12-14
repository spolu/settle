// OWNER stan

package command

import "github.com/spolu/settle/cli"

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
func (c *Pay) Help() {
}

// Parse parses the arguments passed to the command.
func (c *Pay) Parse(
	args []string,
) error {
}

// Execute the command or return a human-friendly error.
func (c *Pay) Execute() error {
}
