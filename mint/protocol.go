package mint

import "math/big"

// AssetResource is the representation of an asset in the mint API.
type AssetResource struct {
	ID       string `json:"id"`
	Created  int64  `json:"created"`
	Livemode bool   `json:"livemode"`

	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Code   string `json:"code"`
	Scale  int8   `json:"scale"`
}

// OperationResource is the representation of an operation in the mint API.
type OperationResource struct {
	ID       string `json:"id"`
	Created  int64  `json:"created"`
	Livemode bool   `json:"livemode"`

	Asset       string  `json:"asset"`
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	Amount      big.Int `json:"amount"`
}
