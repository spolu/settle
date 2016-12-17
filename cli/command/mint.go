// OWNER stan

package command

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

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
	Code  string
	Scale int8
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
	out.Normf("    The asset you want to mint of the form `{code}.{scale}`. The code must be composed\n")
	out.Normf("    of alphanumeric characters or '-'. The scale is an integer between 0 and 24.\n")
	out.Valuf("    USD.2 HOURS-OF-WORK.0 BTC.7 EUR.2 DRINK.0\n")
	out.Normf("\n")
}

var assetRegexp = regexp.MustCompile(
	"^([A-Z0-9-]{1,64})\\.([0-9]{1,2})$",
)

// Parse parses the arguments passed to the command.
func (c *Mint) Parse(
	ctx context.Context,
	args []string,
) error {
	if len(args) == 0 {
		return errors.Trace(
			errors.Newf("Asset name required (see `settle help mint`)"))
	}

	m := assetRegexp.FindStringSubmatch(args[0])
	if len(m) == 0 {
		return errors.Trace(
			errors.Newf("Invalid asset: %s (see `settle help mint`)", args[0]))
	}

	s, err := strconv.ParseInt(m[2], 10, 8)
	if err != nil || s < 0 || s > 24 {
		return errors.Trace(
			errors.Newf(
				"Invalid asset scale: %s (see `settle help mint`)", m[2]))
	}

	c.Code = m[1]
	c.Scale = int8(s)

	return nil
}

// Execute the command or return a human-friendly error.
func (c *Mint) Execute(
	ctx context.Context,
) error {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	out.Statf("[Creating asset] code:%s scale:%d\n",
		c.Code, c.Scale)

	status, raw, err := m.Post(ctx,
		"/assets",
		url.Values{
			"code":  {c.Code},
			"scale": {fmt.Sprintf("%d", c.Scale)},
		})
	if err != nil {
		return errors.Trace(err)
	}

	if *status != http.StatusCreated && *status != http.StatusOK {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var asset mint.AssetResource
	err = raw.Extract("asset", &asset)
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
