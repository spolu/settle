// OWNER stan

package command

import "github.com/spolu/settle/cli"

const (
	// CmdNmMint is the command name.
	CmdNmMint cli.CmdName = "mint"
)

func init() {
	cli.Registrar[CmdNmMint] = NewMint
}

// Mint a user up to a certain amount of a given asset they issued.
type Mint struct {
}

// NewMint constructs and initializes the command.
func NewMint() cli.Command {
	return &Mint{}
}

// Name returns the command name.
func (c *Mint) Name() cli.CmdName {
	return CmdNmMint
}

// Help prints out the help message for the command.
func (c *Mint) Help() {
}

// Parse parses the arguments passed to the command.
func (c *Mint) Parse(
	args []string,
) error {
}

// Execute the command or return a human-friendly error.
func (c *Mint) Execute() error {

}
