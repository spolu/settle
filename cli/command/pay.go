package command

import (
	"bufio"
	"context"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/lib/plan"
)

const (
	// CmdNmPay is the command name.
	CmdNmPay cli.CmdName = "pay"
)

func init() {
	cli.Registrar[CmdNmPay] = NewPay
}

// Candidate represents a candidate base asset, offer path and amount to pay
// the required amount of quote asset.
type Candidate struct {
	Path      []mint.OfferResource
	BaseAsset string
	Amount    big.Int
}

// Candidates is a slice of Candidate implementing sort.Interface
type Candidates []Candidate

// Len implenents the sort.Interface
func (s Candidates) Len() int {
	return len(s)
}

// Less implenents the sort.Interface
func (s Candidates) Less(i, j int) bool {
	return s[i].Amount.Cmp(&s[j].Amount) < 0
}

// Swap implenents the sort.Interface
func (s Candidates) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Pay a user up to a certain amount of a given asset they issued.
type Pay struct {
	QuoteAsset string
	Amount     big.Int
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
func (c *Pay) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle pay <user> <quote_asset> <amount>\n")
	out.Normf("\n")
	out.Normf("  Paying a user with the specified asset (quote_asset) consists in finding a\n")
	out.Normf("  path of trust betwen an asset you own or control (the base asset) and the\n")
	out.Normf("  asset of the payee (quote_asset).\n")
	out.Normf("\n")
	out.Normf("  You can only pay a user in an asset (quote_asset) they have minted.\n")
	out.Normf("\n")
	out.Normf("  Finding a path and a base asset consists in:\n")
	out.Normf("   - retrieving all the assets you control or own a balance in.\n")
	out.Normf("   - compute set of assets trusted by the quote asset.\n")
	out.Normf("   - use these sets to compute a trust path of length 0 or 1.\n")
	out.Normf("\n")
	out.Normf("  Currently, only paths of length at most 1 with one base asset are supported\n")
	out.Normf("  (transactions with longer paths can be created using your mint API directly).\n")
	out.Normf("\n")
	out.Normf("Arguments:\n")
	out.Boldf("  user\n")
	out.Normf("    The user you are paying.\n")
	out.Valuf("    von.neuman@ias.edu elon@settle.network\n")
	out.Normf("\n")
	out.Boldf("  quote_asset\n")
	out.Normf("    The asset to pay the user in of the form `{code}.{scale}`.\n")
	out.Valuf("    USD.2 HOUR-OF-WORK.0 BTC.7 EUR.2 DRINK.0\n")
	out.Normf("\n")
	out.Boldf("  amount\n")
	out.Normf("    The amount of user's asset (quote_asset) that you want to pay. Amount is\n")
	out.Normf("    expressed in the asset you are paying the user with (quote_asset). (USD.2\n")
	out.Normf("    199 represents $1.99, HOUR-OF-WORK.0 1 represents 1 hour of work, and BTC.8\n")
	out.Normf("    252912 represents 0.00252912 BTC).\n")
	out.Valuf("    42\n")
	out.Normf("\n")
	out.Normf("Examples:\n")
	out.Valuf("   setlle pay von.neumann@ias.edu EUR.2 150\n")
	out.Normf("\n")
}

// Parse parses the arguments passed to the command.
func (c *Pay) Parse(
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

	return nil
}

// Execute the command or return a human-friendly error.
func (c *Pay) Execute(
	ctx context.Context,
) error {
	candidates, err := c.ComputeCandidates(ctx)
	if err != nil {
		return errors.Trace(err)
	} else if len(candidates) == 0 {
		return errors.Trace(errors.Newf(
			"No trust path was found to %s for %s.",
			c.QuoteAsset, c.Amount.String()))
	}

	out.Boldf("Candidates:\n")
	for i, c := range candidates {
		if i > 9 {
			break
		}
		out.Normf("  (%d) BaseAsset : ", i)
		out.Valuf("%s\n", c.BaseAsset)
		out.Normf("      Amount    : ")
		out.Valuf("%s\n", c.Amount.String())
		out.Normf("      Path      : ")
		if len(c.Path) == 0 {
			out.Normf("(empty)\n")
		} else {
			for j, o := range c.Path {
				if j > 0 {
					out.Normf("\n                  ")
				}
				out.Valuf("%s", o.Pair)
				out.Normf(" ")
				out.Valuf("%s", o.Price)
			}
			out.Normf("\n")
		}
	}

	reader := bufio.NewReader(os.Stdin)

	out.Normf("Candidate selection [0]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	i := int64(0)
	if choice != "" {
		var err error
		i, err = strconv.ParseInt(choice, 10, 8)
		if err != nil || i < 0 || i >= int64(len(candidates)) {
			return errors.Trace(errors.Newf("Invalid choice: %s", choice))
		}
	}
	candidate := candidates[i]
	_ = candidate

	a, err := mint.AssetResourceFromName(ctx, c.QuoteAsset)
	if err != nil {
		return errors.Trace(err)
	}

	// Ask confirmation for the transaction.
	out.Boldf("Proposed transaction:\n")
	out.Normf("  Destination  : ")
	out.Valuf("%s\n", a.Owner)
	out.Normf("  Pair         : ")
	out.Valuf("%s\n", fmt.Sprintf("%s/%s", candidate.BaseAsset, c.QuoteAsset))
	out.Normf("  You pay      : ")
	out.Valuf("%s %s\n", candidate.BaseAsset, candidate.Amount.String())
	out.Normf("  They receive : ")
	out.Valuf("%s %s\n", c.QuoteAsset, c.Amount.String())
	out.Normf("  Path         : ")
	if len(candidate.Path) == 0 {
		out.Normf("(empty)\n")
	} else {
		for j, o := range candidate.Path {
			if j > 0 {
				out.Normf("\n                 ")
			}
			out.Valuf("%s", o.Pair)
			out.Normf(" ")
			out.Valuf("%s", o.Price)
		}
		out.Normf("\n")
	}

	if err := Confirm(ctx, "pay"); err != nil {
		return errors.Trace(err)
	}

	path := []string{}
	for _, o := range candidate.Path {
		path = append(path, o.ID)
	}

	// Create the transaction.
	tx, err := CreateTransaction(ctx,
		fmt.Sprintf("%s/%s", candidate.BaseAsset, c.QuoteAsset),
		c.Amount, a.Owner, path)
	if err != nil {
		return errors.Trace(err)
	}

	// Settle the transaction.
	tx, err = SettleTransaction(ctx, tx.ID)
	if err != nil {
		return errors.Trace(err)
	}

	out.Boldf("Transaction settled:\n")
	out.Normf("  ID          : ")
	out.Valuf("%s\n", tx.ID)
	out.Normf("  Created     : ")
	out.Valuf("%d\n", tx.Created)
	out.Normf("  Owner       : ")
	out.Valuf("%s\n", tx.Owner)
	out.Normf("  Pair        : ")
	out.Valuf("%s\n", tx.Pair)
	out.Normf("  Amount      : ")
	out.Valuf("%s\n", tx.Amount.String())
	out.Normf("  Destination : ")
	out.Valuf("%s\n", tx.Destination)
	out.Normf("  Path        : ")
	if len(tx.Path) == 0 {
		out.Normf("(empty)\n")
	} else {
		for j, o := range tx.Path {
			if j > 0 {
				out.Normf("\n                ")
			}
			out.Valuf(o)
		}
		out.Normf("\n")
	}
	out.Normf("  Status      : ")
	out.Valuf("%s\n", tx.Status)

	return nil
}

// ComputeCandidates computes candidates to pay the require amount of quote
// asset.
func (c *Pay) ComputeCandidates(
	ctx context.Context,
) (Candidates, error) {
	// Finding a path consists, given the quote asset `q`:
	// - listing all the assets you control or own a balance in: `cSet`
	//   - if `q \in cSet`, use that if possible
	// - compute the set of assets trusted by `q`: `qSet`
	//   - if `cSet \cap qSet` not empty, use that if possible
	// This will generate offer path of length `{0, 1}`.
	g, ctx := errgroup.WithContext(ctx)

	var cSetAssets []mint.AssetResource
	g.Go(func() error {
		var err error
		cSetAssets, err = ListAssets(ctx)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	})

	var cSetBalances []mint.BalanceResource
	g.Go(func() error {
		var err error
		cSetBalances, err = ListBalances(ctx)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	for _, b := range cSetBalances {
		if b.Asset == c.QuoteAsset && b.Value.Cmp(&c.Amount) >= 0 {
			// We own enough of the quote asset itself to pay directly with a
			// path of length 0 and exit early with a unique candidate.
			return Candidates{
				Candidate{
					[]mint.OfferResource{},
					c.QuoteAsset,
					c.Amount,
				},
			}, nil
		}
	}

	// Trigger the retrieval of the qSet to compute path of length 1.
	qSetOffers, err := ListAssetOffers(ctx,
		c.QuoteAsset, mint.PgTpCanonical)
	if err != nil {
		return nil, errors.Trace(err)
	}

	candidates := Candidates{}

	// Compute length 1 candidates from qSet.
	for _, o := range qSetOffers {
		pair, err := mint.AssetResourcesFromPair(ctx, o.Pair)
		if err != nil {
			// ignore error.
			continue
		}

		basePrice, quotePrice, err := plan.ExtractPrice(ctx, o.Price)
		if err != nil {
			// ignore error.
			continue
		}
		amount := new(big.Int).Mul(&c.Amount, basePrice)
		amount, remainder := new(big.Int).QuoRem(
			amount, quotePrice, new(big.Int))

		// Transactions do cross offers on non congruent prices, costing one
		// base unit of quote asset. If the difference of scale between assets
		// is high, this can cost a lot to the owner of the transaction (but if
		// they issued it, they know).
		if remainder.Cmp(big.NewInt(0)) > 0 {
			amount = new(big.Int).Add(amount, big.NewInt(1))
		}

		if o.Remainder.Cmp(&c.Amount) >= 0 {
			for _, b := range cSetBalances {
				if b.Asset == pair[1].Name &&
					b.Value.Cmp(amount) >= 0 {
					candidates = append(candidates, Candidate{
						[]mint.OfferResource{o},
						b.Asset,
						*amount,
					})
				}
			}
			for _, a := range cSetAssets {
				if a.Name == pair[1].Name {
					candidates = append(candidates, Candidate{
						[]mint.OfferResource{o},
						a.Name,
						*amount,
					})
				}
			}
		}
	}

	// Trigger the retrieval of the tSet (asset that trust the cSet) in
	// parrallel to compute paths of length 2.
	// g, ctx = errgroup.WithContext(ctx)
	//
	// tSetOffers := make(
	// 	[][]mint.OfferResource,
	// 	len(cSetAssets)+len(cSetBalances),
	// )
	// g.Go(func() error {
	// 	gT, ctx := errgroup.WithContext(ctx)
	//
	// 	for i, a := range cSetAssets {
	// 		gT.Go(func() error {
	// 			var err error
	// 			tSetOffers[i], _ = ListAssetOffers(ctx,
	// 				a.Name, mint.PgTpPropagated)
	// 			// ignore errors.
	// 			return nil
	// 		})
	// 	}
	// 	for i, b := range cSetBalances {
	// 		gT.Go(func() error {
	// 			var err error
	// 			tSetOffers[len(cSetAssets+i)], _ = ListAssetOffers(ctx,
	// 				b.Asset, mint.PgTpPropagated)
	// 			// ignore errors.
	// 			return nil
	// 		})
	// 	}
	//
	// 	if err := gT.Wait(); err != nil {
	// 		return errors.Trace(err)
	// 	}
	// 	return nil
	// })
	//
	// if err := g.Wait(); err != nil {
	// 	return nil, errors.Trace(err)
	// }

	sort.Sort(candidates)

	return candidates, nil
}
