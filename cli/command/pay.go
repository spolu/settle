package command

import (
	"context"
	"fmt"
	"math/big"

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

// Pay a user up to a certain amount of a given asset they issued.
type Pay struct {
	QuoteAsset  string
	Amount      big.Int
	Destination string
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
			"No turst path was found to %s for %s.",
			c.QuoteAsset, c.Amount.String()))
	}

	// Ask confirmation to use.
	// Create the transaction.
	// Settle the transaction.
	return nil
}

// ComputeCandidates computes candidates to pay the require amount of quote
// asset.
func (c *Pay) ComputeCandidates(
	ctx context.Context,
) ([]Candidate, error) {
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
			return []Candidate{
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

	candidates := []Candidate{}

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
				candidates = append(candidates, Candidate{
					[]mint.OfferResource{o},
					a.Name,
					*amount,
				})
			}
		}
	}

	// Trigger the retrieval of the tSet in parrallel to compute paths of
	// length 2.
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

	return candidates, nil
}
