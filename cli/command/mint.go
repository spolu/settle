// OWNER stan

package command

import (
	"context"
	"fmt"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/mint"
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
	AssetName string
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
	out.Normf("  Minting an asset will enable you to express trust or pay other users. Minting\n")
	out.Normf("  assets is a prerequesite to any other action.\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  asset\n")
	out.Normf("    The asset you want to mint of the form `{code}.{scale}`. The code must be\n")
	out.Normf("    composed of alphanumeric characters or '-'. The scale is an integer between\n")
	out.Normf("    0 and 24. The scale represents the number of decimal used to express asset\n")
	out.Normf("    amounts (USD.2 199 represents $1.99, HOUR-OF-WORK.0 1 represents 1 hour of\n")
	out.Normf("    work, and BTC.8 252912 represents 0.00252912 BTC).\n")
	out.Valuf("    USD.2 HOUR-OF-WORK.0 BTC.7 EUR.2 DRINK.0\n")
	out.Normf("\n")
	out.Normf("Examples:\n")
	out.Valuf("   setlle mint USD.2\n")
	out.Valuf("   setlle mint BTC.7\n")
	out.Valuf("   setlle mint HOUR-Of-WORK.0\n")
	out.Normf("\n")
}

// Parse parses the arguments passed to the command.
func (c *Mint) Parse(
	ctx context.Context,
	args []string,
) error {
	creds := cli.GetCredentials(ctx)
	if creds == nil {
		return errors.Trace(
			errors.Newf("You need to be logged in."))
	}

	if len(args) == 0 {
		return errors.Trace(
			errors.Newf("Asset required."))
	}

	a, err := mint.AssetResourceFromName(ctx,
		fmt.Sprintf("%s@%s[%s]", creds.Username, creds.Host, args[0]))
	if err != nil {
		return errors.Trace(err)
	}

	c.AssetName = a.Name

	return nil
}

// Execute the command or return a human-friendly error.
func (c *Mint) Execute(
	ctx context.Context,
) error {
	asset, err := CreateAsset(ctx, c.AssetName)
	if err != nil {
		return errors.Trace(err)
	}

	out.Boldf("Asset:\n")
	out.Normf("  ID      : ")
	out.Valuf("%s\n", asset.ID)
	out.Normf("  Created : ")
	out.Valuf("%d\n", asset.Created)
	out.Normf("  Owner   : ")
	out.Valuf("%s\n", asset.Owner)
	out.Normf("  Name    : ")
	out.Valuf("%s\n", asset.Name)
	out.Normf("  Code    : ")
	out.Valuf("%s\n", asset.Code)
	out.Normf("  Scale   : ")
	out.Valuf("%d\n", asset.Scale)

	return nil
}
