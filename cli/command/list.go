package command

import (
	"context"
	"fmt"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/mint"
)

// ObjType reperesents a list object type.
type ObjType string

const (
	// CmdNmList is the command name.
	CmdNmList cli.CmdName = "list"

	// ObjTpAsset asset object type.
	ObjTpAsset ObjType = "asset"
	// ObjTpBalance balance object type.
	ObjTpBalance ObjType = "balance"
	// ObjTpTrustline trustline object type.
	ObjTpTrustline ObjType = "trustline"
)

func init() {
	cli.Registrar[CmdNmList] = NewList
}

// List assets, balances, balances for an asset and trustlines.
type List struct {
	Type      ObjType
	AssetName *string
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
	out.Boldf("settle list <type> [<asset>]\n")
	out.Normf("\n")
	out.Normf("  Lists assets, balances (yours or related to one of your assets) or trustlines\n")
	out.Normf("  (from you, and to you for a particular asset).\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  type\n")
	out.Normf("    The type of object to retrieve and list.\n")
	out.Valuf("    assets balances trustlines\n")
	out.Normf("\n")
	out.Boldf("  asset\n")
	out.Normf("    Applicable for balances and required for trustlines. If used with balances,\n")
	out.Normf("    list all the balances for one of your asset (all other users' balances);\n")
	out.Normf("    when used with trustlines, list all the trustlines for a particular asset.\n")
	out.Valuf("    USD.2 HOUR-OF-WORK.0 BTC.7 EUR.2 DRINK.0\n")
	out.Normf("\n")
	out.Normf("Examples:\n")
	out.Valuf("   setlle list assets\n")
	out.Valuf("   setlle list balances\n")
	out.Valuf("   setlle list balances USD.2\n")
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
			errors.Newf("You need to be logged in (try `settle help login`)."))
	}

	if len(args) == 0 {
		return errors.Trace(
			errors.Newf("Object required (assets, balances, or trustlines)."))
	}
	typ, args := args[0], args[1:]

	switch typ {
	case "assets", "asset":
		c.Type = ObjTpAsset
	case "balances", "balance":
		c.Type = ObjTpBalance
	case "trustlines", "trustline", "trusts", "trust":
		c.Type = ObjTpTrustline
	default:
		return errors.Trace(
			errors.Newf("Invalid object type: %s expected assets balances, "+
				"or trustlines.", typ))
	}

	if len(args) > 0 {
		asset := args[0]

		switch c.Type {
		case ObjTpBalance, ObjTpTrustline:
			a, err := mint.AssetResourceFromName(ctx,
				fmt.Sprintf("%s@%s[%s]", creds.Username, creds.Host, asset))
			if err != nil {
				return errors.Trace(err)
			}
			c.AssetName = &a.Name
		}
	} else {
		switch c.Type {
		case ObjTpTrustline:
			return errors.Trace(
				errors.Newf("Asset required."))
		}
	}

	return nil
}

// Execute the command or return a human-friendly error.
func (c *List) Execute(
	ctx context.Context,
) error {
	switch c.Type {
	case ObjTpAsset:
		return c.ExecuteAssets(ctx)
	case ObjTpBalance:
		return c.ExecuteBalances(ctx)
	case ObjTpTrustline:
		return c.ExecuteTrustlines(ctx)
	}
	return nil
}

// OutList prints out a list of records.
func (c *List) OutList(
	ctx context.Context,
	list [][][2]string,
) error {
	for _, d := range list {
		out.Normf(" ")
		for _, v := range d {
			out.Normf(" %s: ", v[0])
			out.Valuf("%s", v[1])
		}
		out.Normf("\n")
	}
	return nil
}

// ExecuteAssets the list command for assets.
func (c *List) ExecuteAssets(
	ctx context.Context,
) error {
	assets, err := ListAssets(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	out.Boldf("Assets:\n")
	data := [][][2]string{}
	for _, a := range assets {
		data = append(data, [][2]string{
			[2]string{"ID", a.ID},
			[2]string{"Created", fmt.Sprintf("%d", a.Created)},
			[2]string{"Asset", a.Name},
		})
	}
	if len(assets) == 0 {
		out.Normf("  No asset.\n")
	} else {
		c.OutList(ctx, data)
	}

	return nil
}

// ExecuteBalances the list command for balances.
func (c *List) ExecuteBalances(
	ctx context.Context,
) error {
	var balances []mint.BalanceResource
	var err error
	if c.AssetName == nil {
		balances, err = ListBalances(ctx)
		if err != nil {
			return errors.Trace(err)
		}
	} else {
		balances, err = ListAssetBalances(ctx, *c.AssetName)
		if err != nil {
			return errors.Trace(err)
		}
	}

	out.Boldf("Balances:\n")
	data := [][][2]string{}
	for _, b := range balances {
		data = append(data, [][2]string{
			[2]string{"ID", b.ID},
			[2]string{"Created", fmt.Sprintf("%d", b.Created)},
			[2]string{"Asset", b.Asset},
			[2]string{"Holder", b.Holder},
			[2]string{"Value", b.Value.String()},
		})
	}
	if len(balances) == 0 {
		out.Normf("  No balance.\n")
	} else {
		c.OutList(ctx, data)
	}

	return nil
}

// ExecuteTrustlines the list command for balances.
func (c *List) ExecuteTrustlines(
	ctx context.Context,
) error {
	cOffers, err := ListAssetOffers(ctx, *c.AssetName, mint.PgTpCanonical)
	if err != nil {
		return errors.Trace(err)
	}
	pOffers, err := ListAssetOffers(ctx, *c.AssetName, mint.PgTpPropagated)
	if err != nil {
		return errors.Trace(err)
	}

	out.Boldf("Trustlines from you:\n")
	data := [][][2]string{}
	for _, o := range cOffers {
		data = append(data, [][2]string{
			[2]string{"ID", o.ID},
			[2]string{"Created", fmt.Sprintf("%d", o.Created)},
			[2]string{"Pair", o.Pair},
			[2]string{"Price", o.Price},
			[2]string{"Amount", o.Amount.String()},
			[2]string{"Status", string(o.Status)},
			[2]string{"Remainder", o.Remainder.String()},
		})
	}
	if len(cOffers) == 0 {
		out.Normf("  No trustline.\n")
	} else {
		c.OutList(ctx, data)
	}

	out.Boldf("Trustlines to you:\n")
	data = [][][2]string{}
	for _, o := range pOffers {
		data = append(data, [][2]string{
			[2]string{"ID", o.ID},
			[2]string{"Created", fmt.Sprintf("%d", o.Created)},
			[2]string{"Pair", o.Pair},
			[2]string{"Price", o.Price},
			[2]string{"Amount", o.Amount.String()},
			[2]string{"Status", string(o.Status)},
			[2]string{"Remainder", o.Remainder.String()},
		})
	}
	if len(pOffers) == 0 {
		out.Normf("  No trustline.\n")
	} else {
		c.OutList(ctx, data)
	}

	return nil
}
