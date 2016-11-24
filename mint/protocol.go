package mint

import (
	"context"
	"fmt"
	"math/big"

	"github.com/spolu/settle/mint/model"
)

const (
	// Version is the current protocol version.
	Version string = "0"
)

// AssetResource is the representation of an asset in the mint API.
type AssetResource struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Owner   string `json:"owner"`

	Name  string `json:"name"`
	Code  string `json:"code"`
	Scale int8   `json:"scale"`
}

// NewAssetResource generates a new resource.
func NewAssetResource(
	ctx context.Context,
	asset *model.Asset,
) AssetResource {
	return AssetResource{
		ID: fmt.Sprintf(
			"%s[%s]", asset.Owner, asset.Token),
		Created: asset.Created.UnixNano() / (1000 * 1000),
		Owner:   asset.Owner,
		Name: fmt.Sprintf(
			"%s[%s.%d]",
			asset.Owner, asset.Code, asset.Scale,
		),
		Code:  asset.Code,
		Scale: asset.Scale,
	}
}

// OperationResource is the representation of an operation in the mint API.
type OperationResource struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Owner   string `json:"owner"`

	Asset       AssetResource `json:"asset"`
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
	Amount      *big.Int      `json:"amount"`
}

// NewOperationResource generates a new resource.
func NewOperationResource(
	ctx context.Context,
	operation *model.Operation,
	asset *model.Asset,
) OperationResource {
	return OperationResource{
		ID: fmt.Sprintf(
			"%s[%s]", operation.Owner, operation.Token),
		Created:     operation.Created.UnixNano() / (1000 * 1000),
		Owner:       operation.Owner,
		Asset:       NewAssetResource(ctx, asset),
		Source:      operation.Source,
		Destination: operation.Destination,
		Amount:      (*big.Int)(&operation.Amount),
	}
}

// OfferResource is the representation of an offer in the mint API.
type OfferResource struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Owner   string `json:"owner"`

	Pair   string   `json:"pair"`
	Price  string   `json:"price"`
	Amount *big.Int `json:"amount"`
	Status string   `json:"status"`
}

// NewOfferResource generates a new resource.
func NewOfferResource(
	ctx context.Context,
	offer *model.Offer,
) OfferResource {
	return OfferResource{
		ID: fmt.Sprintf(
			"%s[%s]", offer.Owner, offer.Token),
		Created: offer.Created.UnixNano() / (1000 * 1000),
		Owner:   offer.Owner,
		Pair:    fmt.Sprintf("%s/%s", offer.BaseAsset, offer.QuoteAsset),
		Price: fmt.Sprintf(
			"%s/%s",
			(*big.Int)(&offer.BasePrice).String(),
			(*big.Int)(&offer.QuotePrice).String()),
		Amount: (*big.Int)(&offer.Amount),
		Status: string(offer.Status),
	}
}

// TransactionResource is the representation of a transaction in the mint API.
type TransactionResource struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Owner   string `json:"owner"`

	Pair        string   `json:"pair"`
	Amount      *big.Int `json:"amount"`
	Destination string   `json:"destination"`
	Path        []string `json:"path"`
}

// NewTransactionResource generates a new resource.
func NewTransactionResource(
	ctx context.Context,
	transaction *model.Transaction,
) TransactionResource {
	return TransactionResource{
		ID: fmt.Sprintf(
			"%s[%s]", transaction.Owner, transaction.Token),
		Created: transaction.Created.UnixNano() / (1000 * 1000),
		Owner:   transaction.Owner,
		Pair: fmt.Sprintf("%s/%s",
			transaction.BaseAsset, transaction.QuoteAsset),
		Amount:      (*big.Int)(&transaction.Amount),
		Destination: transaction.Destination,
		Path:        []string(transaction.Path),
	}
}
