package command

import (
	"context"
	"fmt"
	"math/big"
	"regexp"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/mint"
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
	BaseAsset  string
	QuoteAsset string
	Amount     big.Int
	Price      string
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
	out.Normf("\nUsage: ")
	out.Boldf("settle trust <user> <quote_asset> <amount> [with <base_asset> at <price>]\n")
	out.Normf("\n")
	out.Normf("  Trusting user's asset (quote_asset) expresses your commitment to issue your\n")
	out.Normf("  own asset (base_asset) in exchange for the quote asset at the specified\n")
	out.Normf("  exchange price and up to the specified amount.\n")
	out.Normf("\n")
	out.Normf("  Once your tustline is created, anyone will be able to credit you with user's\n")
	out.Normf("  asset (quote_asset) in exchange for your own asset (base_asset). Your mint\n")
	out.Normf("  will automatically issue the required amount of your asset (base_asset) to\n")
	out.Normf("  satisfy the exchange at the specified price. Your mint will stop accepting\n")
	out.Normf("  exchanges or issuing your asset once the trustline is consumed (you exchanged\n")
	out.Normf("  all of the specified amount in one or many transactions).\n")
	out.Normf("\n")
	out.Normf("  The last two arguments can be ommitted in which case: the same asset code and\n")
	out.Normf("  scale will be used for your asset (which requires that you have minted that\n")
	out.Normf("  asset); the price 1/1 will be used by default (exchange at parity).\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  user\n")
	out.Normf("    The user you are committing to trust.\n")
	out.Valuf("    von.neuman@ias.edu elon@settle.network\n")
	out.Normf("\n")
	out.Boldf("  quote_asset\n")
	out.Normf("    The asset from user that you are committing to trust of the form\n")
	out.Normf("    `{code}.{scale}`.\n")
	out.Valuf("    USD.2 HOUR-OF-WORK.0 BTC.7 EUR.2 DRINK.0\n")
	out.Normf("\n")
	out.Boldf("  amount\n")
	out.Normf("    The amount of user's asset (quote_asset) that you are committing to trust.\n")
	out.Normf("    Amount is expressed in the asset you are trusting using the asset scale\n")
	out.Normf("    (USD.2 199 represents $1.99, HOUR-OF-WORK.0 1 represents 1 hour of work,\n")
	out.Normf("    and BTC.8 252912 represents 0.00252912 BTC).\n")
	out.Valuf("    42\n")
	out.Normf("\n")
	out.Boldf("  base_asset\n")
	out.Normf("    One of your assets that you are committing to issue in exchange for user's\n")
	out.Normf("    asset, of the form `{code}.{scale}`. If the asset was not minted previously\n")
	out.Normf("    it will be created.\n")
	out.Valuf("    USD.2 HOUR-OF-WORK.0 BTC.7 EUR.2 DRINK.0\n")
	out.Normf("\n")
	out.Boldf("  price\n")
	out.Normf("    The price at which you are committing to exchange the user asset for your\n")
	out.Normf("    asset. The price is expressed in {base_asset}/{quote_asset}.\n")
	out.Valuf("    1/1 100/125 4500/1\n")
	out.Normf("\n")
	out.Normf("Examples:\n")
	out.Valuf("   setlle trust von.neumann@ias.edu USD.2 150\n")
	out.Valuf("   setlle trust kurt@princetown.edu USD.2 150 with USD.2 at 1/1\n")
	out.Valuf("   setlle trust alan@npl.co.uk GBP.2 120 with USD.2 at 125/100\n")
	out.Valuf("   setlle trust venture@risky.co USD.2 1200 with USD.2 at 100/75\n")
	out.Normf("\n")
}

// PriceRegexp is used to validate and parse a price.
var PriceRegexp = regexp.MustCompile(
	"^([0-9]+)\\/([0-9]+)$")

// Parse parses the arguments passed to the command.
func (c *Trust) Parse(
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
			errors.Newf("User required."))
	}
	user, args := args[0], args[1:]

	_, _, err := mint.UsernameAndMintHostFromAddress(ctx, user)
	if err != nil {
		return errors.Trace(err)
	}

	if len(args) == 0 {
		return errors.Trace(
			errors.Newf("Quote asset required."))
	}
	asset, args := args[0], args[1:]

	qA, err := mint.AssetResourceFromName(ctx,
		fmt.Sprintf("%s[%s]", user, asset))
	if err != nil {
		return errors.Trace(err)
	}
	c.QuoteAsset = qA.Name

	if len(args) == 0 {
		return errors.Trace(
			errors.Newf("Amount required."))
	}
	amount, args := args[0], args[1:]

	var amt big.Int
	if _, success := amt.SetString(amount, 10); !success {
		return errors.Newf("Invalid amount: %s", amount)
	}
	c.Amount = amt

	// Accept `with`.
	if len(args) > 0 && args[0] == "with" {
		args = args[1:]
	}

	// Default `base_asset` and `price`.
	if len(args) == 0 {
		c.BaseAsset = fmt.Sprintf("%s@%s[%s.%d]",
			creds.Username, creds.Host, qA.Code, qA.Scale)
		c.Price = "1/1"
		return nil
	}

	asset, args = args[0], args[1:]
	bA, err := mint.AssetResourceFromName(ctx,
		fmt.Sprintf("%s@%s[%s]", creds.Username, creds.Host, asset))
	if err != nil {
		return errors.Trace(err)
	}
	c.BaseAsset = bA.Name

	// Accept `at`.
	if len(args) > 0 && args[0] == "at" {
		args = args[1:]
	}

	// Default `price`.
	if len(args) == 0 {
		c.Price = "1/1"
		return nil
	}

	price, args := args[0], args[1:]
	m := PriceRegexp.FindStringSubmatch(price)
	if len(m) == 0 {
		return errors.Trace(errors.Newf("Invalid price: %s", price))
	}
	c.Price = price

	return nil
}

// Execute the command or return a human-friendly error.
func (c *Trust) Execute(
	ctx context.Context,
) error {
	// Retrieve base asset to check existence
	asset, err := RetrieveAsset(ctx, c.BaseAsset)
	if err != nil {
		return errors.Trace(err)
	} else if asset == nil {
		bA, err := mint.AssetResourceFromName(ctx, c.BaseAsset)
		if err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(
			errors.Newf("You need to mint %s.%d first "+
				"(see `settle help mint`).",
				bA.Code, bA.Scale))
	}
	// Retrieve quote asset to check existence
	asset, err = RetrieveAsset(ctx, c.QuoteAsset)
	if err != nil {
		return errors.Trace(err)
	} else if asset == nil {
		qA, err := mint.AssetResourceFromName(ctx, c.QuoteAsset)
		if err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(
			errors.Newf("%s does not exists, ask %s to mint %s.%d "+
				"first (see `settle help mint`).",
				qA.Name, qA.Owner, qA.Code, qA.Scale))
	}

	out.Boldf("Proposed trustline:\n")
	out.Normf("  Pair      : ")
	out.Valuf("%s\n", fmt.Sprintf("%s/%s", c.BaseAsset, c.QuoteAsset))
	out.Normf("  Price     : ")
	out.Valuf("%s\n", c.Price)
	out.Normf("  Amount    : ")
	out.Valuf("%s\n", c.Amount.String())

	if err := Confirm(ctx, "trust"); err != nil {
		return errors.Trace(err)
	}

	// Create offer.
	offer, err := CreateOffer(ctx,
		fmt.Sprintf("%s/%s", c.BaseAsset, c.QuoteAsset),
		c.Amount,
		c.Price,
	)
	if err != nil {
		return errors.Trace(err)
	}

	out.Boldf("Trustline created:\n")
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
