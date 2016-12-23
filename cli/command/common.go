package command

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/client"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/lib/svc"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/register"
)

// MintRegister contains all the required information to register to a mint
// from the cli.
type MintRegister struct {
	Name        string
	Host        string
	Description string
	RegisterURL map[env.Environment]string
}

// PublicMints is a list of proposed public mints that offer registration
// form the cli.
var PublicMints = []MintRegister{
	MintRegister{
		Name:        "Settle",
		Host:        "settle.network",
		Description: "Mint maintained by the Settle developers.",
		RegisterURL: map[env.Environment]string{
			env.Production: "https://register.settle.network/users",
			env.QA:         "https://qa-register.settle.network/users",
		},
	},
}

// RegisterUser registers a user on the provded mint register service.
func RegisterUser(
	ctx context.Context,
	reg MintRegister,
	username string,
	email string,
) (*register.UserResource, error) {
	body := url.Values{}
	body["username"] = []string{username}
	body["email"] = []string{email}

	req, err := http.NewRequest("POST",
		reg.RegisterURL[env.Get(ctx).Environment], nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r, err := client.Default(ctx).Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer r.Body.Close()

	var raw svc.Resp
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.Trace(err)
	}

	if r.StatusCode != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var user register.UserResource
	err = raw.Extract("user", &user)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &user, nil
}

// CreateAsset creates an asset for the currently authenticated user.
func CreateAsset(
	ctx context.Context,
	name string,
) (*mint.AssetResource, error) {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	out.Statf("[Creating asset] user=%s@%s asset=%s\n",
		m.Credentials.Username, m.Credentials.Host,
		name)

	a, err := mint.AssetResourceFromName(ctx, name)
	if err != nil {
		return nil, errors.Trace(err)
	}

	status, raw, err := m.Post(ctx,
		"/assets",
		url.Values{},
		url.Values{
			"code":  {a.Code},
			"scale": {fmt.Sprintf("%d", a.Scale)},
		})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if *status != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var asset mint.AssetResource
	err = raw.Extract("asset", &asset)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &asset, nil
}

// RetrieveAsset retrieves an asset, returnin nil if it does not exist.
func RetrieveAsset(
	ctx context.Context,
	name string,
) (*mint.AssetResource, error) {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	out.Statf("[Retrieving asset] user=%s@%s asset=%s\n",
		m.Credentials.Username, m.Credentials.Host,
		name)

	status, raw, err := m.Get(ctx,
		fmt.Sprintf("/assets/%s", name),
		url.Values{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if *status != http.StatusOK {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if e.ErrCode == "asset_not_found" {
			return nil, nil
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var asset mint.AssetResource
	err = raw.Extract("asset", &asset)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &asset, nil
}

// CreateOffer creates an offer for the currently authenticated user.
func CreateOffer(
	ctx context.Context,
	pair string,
	amount big.Int,
	price string,
) (*mint.OfferResource, error) {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	out.Statf("[Creating offer] user=%s@%s pair=%s amount=%s price=%s\n",
		m.Credentials.Username, m.Credentials.Host,
		pair, amount.String(), price)

	status, raw, err := m.Post(ctx,
		"/offers",
		url.Values{},
		url.Values{
			"pair":   {pair},
			"amount": {amount.String()},
			"price":  {price},
		})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if *status != http.StatusCreated {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var offer mint.OfferResource
	err = raw.Extract("offer", &offer)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &offer, nil
}

// ListAssets list assets for the current user.
func ListAssets(
	ctx context.Context,
) ([]mint.AssetResource, error) {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	out.Statf("[Listing assets] user=%s@%s\n",
		m.Credentials.Username, m.Credentials.Host)

	status, raw, err := m.Get(ctx,
		"/assets",
		url.Values{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if *status != http.StatusOK {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var assets []mint.AssetResource
	err = raw.Extract("assets", &assets)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return assets, nil
}

// ListBalances list balances of the current user.
func ListBalances(
	ctx context.Context,
) ([]mint.BalanceResource, error) {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	out.Statf("[Listing balances] user=%s@%s\n",
		m.Credentials.Username, m.Credentials.Host)

	status, raw, err := m.Get(ctx,
		"/balances",
		url.Values{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if *status != http.StatusOK {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var balances []mint.BalanceResource
	err = raw.Extract("balances", &balances)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return balances, nil
}

// ListAssetBalances list balances for one of the current user's asset.
func ListAssetBalances(
	ctx context.Context,
	asset string,
) ([]mint.BalanceResource, error) {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	out.Statf("[Listing asset balances] user=%s@%s asset=%s\n",
		m.Credentials.Username, m.Credentials.Host, asset)

	status, raw, err := m.Get(ctx,
		fmt.Sprintf("/assets/%s/balances", asset),
		url.Values{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if *status != http.StatusOK {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var balances []mint.BalanceResource
	err = raw.Extract("balances", &balances)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return balances, nil
}

// ListAssetOffers list offers for one of the current user's asset.
func ListAssetOffers(
	ctx context.Context,
	asset string,
	propagation mint.PgType,
) ([]mint.OfferResource, error) {
	m, err := cli.MintFromContextCredentials(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	out.Statf("[Listing asset offers] user=%s@%s asset=%s propagation=%s\n",
		m.Credentials.Username, m.Credentials.Host, asset, propagation)

	status, raw, err := m.Get(ctx,
		fmt.Sprintf("/assets/%s/offers", asset),
		url.Values{
			"propagation": {string(propagation)},
		})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if *status != http.StatusOK {
		var e errors.ConcreteUserError
		err = raw.Extract("error", &e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return nil, errors.Trace(
			errors.Newf("(%s) %s", e.ErrCode, e.ErrMessage))
	}

	var offers []mint.OfferResource
	err = raw.Extract("offers", &offers)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return offers, nil
}
