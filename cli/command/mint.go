// OWNER stan

package command

import (
	"context"
	"regexp"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
)

const (
	// CmdNmMint is the command name.
	CmdNmMint cli.CmdName = "mint"
)

func init() {
	cli.Registrar[CmdNmMint] = NewMint
}

// Mint a user up to a certain amount of a given asset they issued.
type Mint struct {
	Asset string
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
func (c *Mint) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle mint <asset>\n")
	out.Normf("\n")
	out.Normf("  Minting an asset will create it on your mint allowing you to express trust or pay\n")
	out.Normf("  other users. Minting assets is a prerequesite to any other action.\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  asset\n")
	out.Normf("    The asset you want to mint of the form {CODE}.{SCALE}\n")
	out.Valuf("    USD.2\n")
	out.Normf("\n")
}

var assetRegexp = regexp.MustCompile(
	"([A-Z0-9-]{1,64})\\.([0-9]{1,2})",
)

// Parse parses the arguments passed to the command.
func (c *Mint) Parse(
	ctx context.Context,
	args []string,
) error {
	if len(args) == 0 {
		return errors.Trace(errors.Newf("Asset name required"))
	}
	if !assetRegexp.MatchString(args[0]) {
		return errors.Trace(errors.Newf("Invalid asset: %s", args[0]))
	}

	c.Asset = args[0]

	return nil
}

// Execute the command or return a human-friendly error.
func (c *Mint) Execute(
	ctx context.Context,
) error {
	return nil
}
