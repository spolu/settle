// OWNER stan

package command

import (
	"context"
	"math/big"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
)

const (
	// CmdNmList is the command name.
	CmdNmList cli.CmdName = "list"
)

func init() {
	cli.Registrar[CmdNmList] = NewList
}

// List assets, balances, balances for an asset and trustlines.
type List struct {
	BaseAsset  string
	QuoteAsset string
	Amount     big.Int
	Price      string
}

// NewList constructs and initializes the command.
func NewList() cli.Command {
	return &List{}
}

// Name returns the command name.
func (c *List) Name() cli.CmdName {
	return CmdNmList
}

// Help prints out the help message for the command.
func (c *List) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle list <object> [<asset>]\n")
	out.Normf("\n")
	out.Normf("  Lists assets, balances (yours or related to one of your assets), trustlines\n")
	out.Normf("  (from you and to you for a particular asset).\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  object\n")
	out.Normf("    The type of object to retrieve and list.\n")
	out.Valuf("    assets balances trustlines\n")
	out.Normf("\n")
	out.Boldf("  asset\n")
	out.Normf("    Applicable for balances and trustlines. If used with balances, list all the\n")
	out.Normf("    balances for one of your asset (all other users' balances); if used with\n")
	out.Normf("    trustlines, list all the trustlines for a particular asset.\n")
	out.Valuf("    USD.2 HOUR-OF-WORK.0 BTC.7 EUR.2 DRINK.0\n")
	out.Normf("\n")
	out.Normf("Examples:\n")
	out.Valuf("   setlle list assets\n")
	out.Valuf("   setlle list balances\n")
	out.Valuf("   setlle list balances USD.2\n")
	out.Valuf("   setlle list trustlines\n")
	out.Valuf("   setlle list trustlines EUR.2\n")
	out.Normf("\n")
}

// Parse parses the arguments passed to the command.
func (c *List) Parse(
	ctx context.Context,
	args []string,
) error {
	creds := cli.GetCredentials(ctx)
	if creds == nil {
		return errors.Trace(
			errors.Newf("You need to be logged in (try `settle help login`."))
	}
	return nil
}

// Execute the command or return a human-friendly error.
func (c *List) Execute(
	ctx context.Context,
) error {
	return nil
}
