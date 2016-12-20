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
	out.Normf("  Lists assets, balances (yours or related to one of your assets), trustlines\n")
	out.Normf("  (from you and to you for a particular asset).\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  type\n")
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
	list []map[string]string,
) error {
	for _, d := range list {
		out.Normf(" ")
		for k, v := range d {
			out.Normf(" %s: ", k)
			out.Valuf("%s", v)
		}
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
	data := []map[string]string{}
	for _, a := range assets {
		data = append(data, map[string]string{
			"ID":      a.ID,
			"Created": fmt.Sprintf("%d", a.Created),
			"Asset":   a.Name,
		})
	}
	if len(assets) == 0 {
		out.Normf("No asset.")
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
	data := []map[string]string{}
	for _, b := range balances {
		data = append(data, map[string]string{
			"ID":      b.ID,
			"Created": fmt.Sprintf("%d", b.Created),
			"Asset":   b.Asset,
			"Holder":  b.Holder,
			"Value":   b.Value.String(),
		})
	}
	if len(balances) == 0 {
		out.Normf("No balance.")
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
	data := []map[string]string{}
	for _, o := range cOffers {
		data = append(data, map[string]string{
			"ID":        o.ID,
			"Created":   fmt.Sprintf("%d", o.Created),
			"Pair":      o.Pair,
			"Price":     o.Price,
			"Amount":    o.Amount.String(),
			"Status":    string(o.Status),
			"Remainder": o.Remainder.String(),
		})
	}
	if len(cOffers) == 0 {
		out.Normf("No trustline.\n")
	} else {
		c.OutList(ctx, data)
	}

	out.Boldf("Trustlines to you:\n")
	data = []map[string]string{}
	for _, o := range pOffers {
		data = append(data, map[string]string{
			"ID":        o.ID,
			"Created":   fmt.Sprintf("%d", o.Created),
			"Pair":      o.Pair,
			"Price":     o.Price,
			"Amount":    o.Amount.String(),
			"Status":    string(o.Status),
			"Remainder": o.Remainder.String(),
		})
	}
	if len(pOffers) == 0 {
		out.Normf("No trustline.\n")
	} else {
		c.OutList(ctx, data)
	}

	return nil
}
