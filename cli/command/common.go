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

	out.Statf("[Creating asset] asset=%s\n", name)

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

	out.Statf("[Retrieving asset] asset=%s\n", name)

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

	out.Statf("[Creating offer] pair=%s amount=%s price=%s\n",
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
