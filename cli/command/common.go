package command

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
	"github.com/spolu/settle/mint"
)

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
		fmt.Sprintf("/assets/%s", name))
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

	status, raw, err := m.Get(ctx, "/assets")
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

	status, raw, err := m.Get(ctx, "/balances")
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

	status, raw, err := m.Get(ctx, fmt.Sprintf("/assets/%s/balances", asset))
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
		fmt.Sprintf("/assets/%s/offers?propagation=%s", asset, propagation))
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
