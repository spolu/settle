package command

import (
	"context"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/mint"
)

const (
	// CmdNmClose is the command name.
	CmdNmClose cli.CmdName = "close"
)

func init() {
	cli.Registrar[CmdNmClose] = NewClose
}

// Close close an existing trustline.
type Close struct {
	ID string
}

// NewClose constructs and initializes the command.
func NewClose() cli.Command {
	return &Close{}
}

// Name returns the command name.
func (c *Close) Name() cli.CmdName {
	return CmdNmClose
}

// Help prints out the help message for the command.
func (c *Close) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle close <trustline>\n")
	out.Normf("\n")
	out.Normf("  Closing a trustline prevents any future use of it.\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  trustline\n")
	out.Normf("    The ID of the trustline to close as returned by the ")
	out.Boldf("list")
	out.Normf(" command.\n")
	out.Valuf("    spolu@m.settle.network[offer_Z1vJtLYFUFgSWwy0]\n")
	out.Normf("\n")
	out.Normf("Examples:\n")
	out.Valuf("  settle close spolu@m.settle.network[offer_Z1vJtLYFUFgSWwy0]\n")
	out.Normf("\n")
}

// Parse parses the arguments passed to the command.
func (c *Close) Parse(
	ctx context.Context,
	args []string,
) error {
	creds := cli.GetCredentials(ctx)
	if creds == nil {
		return errors.Trace(
			errors.Newf("You need to be logged in (try `settle help login`)."))
	}

	if len(args) == 0 {
		return errors.Trace(
			errors.Newf("Trustline ID required."))
	}
	id, args := args[0], args[1:]

	_, _, err := mint.NormalizedOwnerAndTokenFromID(ctx, id)
	if err != nil {
		return errors.Trace(err)
	}

	c.ID = id

	return nil
}

// Execute the command or return a human-friendly error.
func (c *Close) Execute(
	ctx context.Context,
) error {
	// Retrieve offer to check existence
	offer, err := RetrieveOffer(ctx, c.ID)
	if err != nil {
		return errors.Trace(err)
	} else if offer == nil {
		return errors.Trace(
			errors.Newf("Offer does not exists."))
	}

	out.Boldf("Trustline to close:\n")
	out.Normf("  ID        : ")
	out.Valuf("%s\n", offer.ID)
	out.Normf("  Created   : ")
	out.Valuf("%d\n", offer.Created)
	out.Normf("  Owner     : ")
	out.Valuf("%s\n", offer.Owner)
	out.Normf("  Pair      : ")
	out.Valuf("%s\n", offer.Pair)
	out.Normf("  Price     : ")
	out.Valuf("%s\n", offer.Price)
	out.Normf("  Amount    : ")
	out.Valuf("%s\n", offer.Amount.String())
	out.Normf("  Status    : ")
	out.Valuf("%s\n", offer.Status)
	out.Normf("  Remainder : ")
	out.Valuf("%s\n", offer.Remainder.String())

	if err := Confirm(ctx, "close"); err != nil {
		return errors.Trace(err)
	}

	offer, err = CloseOffer(ctx, c.ID)
	if err != nil {
		return errors.Trace(err)
	}

	out.Boldf("Trustline closed:\n")
	out.Normf("  ID        : ")
	out.Valuf("%s\n", offer.ID)
	out.Normf("  Created   : ")
	out.Valuf("%d\n", offer.Created)
	out.Normf("  Owner     : ")
	out.Valuf("%s\n", offer.Owner)
	out.Normf("  Pair      : ")
	out.Valuf("%s\n", offer.Pair)
	out.Normf("  Price     : ")
	out.Valuf("%s\n", offer.Price)
	out.Normf("  Amount    : ")
	out.Valuf("%s\n", offer.Amount.String())
	out.Normf("  Status    : ")
	out.Valuf("%s\n", offer.Status)
	out.Normf("  Remainder : ")
	out.Valuf("%s\n", offer.Remainder.String())

	return nil
}
