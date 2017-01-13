package mint

import "math/big"

const (
	// ProtocolVersion is the current protocol version.
	ProtocolVersion string = "0"
	// TimeResolutionNs is the resolution of our time variables in nanoseconds
	// (aka resolution in milliseconds).
	TimeResolutionNs int64 = 1000 * 1000
	// TransactionExpiryMs is the time it takes to attempt to cancel a
	// transaction for this mint. Expressed in ms.
	TransactionExpiryMs int64 = 1000 * 60 * 60
)

// PgType is the propagation type of an object.
type PgType string

const (
	// PgTpCanonical is an offer owned by this mint.
	PgTpCanonical PgType = "canonical"
	// PgTpPropagated is an offer propagated to this mint.
	PgTpPropagated PgType = "propagated"
)

// OfStatus is the status of an offer.
type OfStatus string

const (
	// OfStActive is used to mark an offer as active.
	OfStActive OfStatus = "active"
	// OfStClosed is used to mark an offer as closed.
	OfStClosed OfStatus = "closed"
	// OfStConsumed is used to mark an offer as consumed.
	OfStConsumed OfStatus = "consumed"
)

// TxStatus is the status of a transaction, operation or crossing.
type TxStatus string

const (
	// TxStPending is used to mark a transaction as pending (created but not
	// yet finalized)
	TxStPending TxStatus = "pending"
	// TxStReserved is used to mark a transaction or action (operation or
	// crossing) as reserved.
	TxStReserved TxStatus = "reserved"
	// TxStSettled is used to mark a transaction or action (operation or
	// crossing) as settled.
	TxStSettled TxStatus = "settled"
	// TxStCanceled is used to mark a transaction or action (operation or
	// crossing) as canceled.
	TxStCanceled TxStatus = "canceled"
)

// AssetResource is the representation of an asset in the mint API.
type AssetResource struct {
	ID          string `json:"id"`
	Created     int64  `json:"created"`
	Owner       string `json:"owner"`
	Propagation PgType `json:"propagation"`

	Name  string `json:"name"`
	Code  string `json:"code"`
	Scale int8   `json:"scale"`
}

// BalanceResource is the representation of an asset balance in the mint API.
type BalanceResource struct {
	ID          string `json:"id"`
	Created     int64  `json:"created"`
	Owner       string `json:"owner"`
	Propagation PgType `json:"propagation"`

	Asset  string   `json:"asset"`
	Holder string   `json:"holder"`
	Value  *big.Int `json:"value"`
}

// OperationResource is the representation of an operation in the mint API.
type OperationResource struct {
	ID          string `json:"id"`
	Created     int64  `json:"created"`
	Owner       string `json:"owner"`
	Propagation PgType `json:"propagation"`

	Asset       string   `json:"asset"`
	Source      string   `json:"source"`
	Destination string   `json:"destination"`
	Amount      *big.Int `json:"amount"`

	Status         TxStatus `json:"status"`
	Transaction    *string  `json:"transaction"`
	TransactionHop *int8    `json:"transaction_hop"`
}

// OfferResource is the representation of an offer in the mint API.
type OfferResource struct {
	ID          string `json:"id"`
	Created     int64  `json:"created"`
	Owner       string `json:"owner"`
	Propagation PgType `json:"propagation"`

	Pair   string   `json:"pair"`
	Price  string   `json:"price"`
	Amount *big.Int `json:"amount"`

	Status    OfStatus `json:"status"`
	Remainder *big.Int `json:"remainder"`
}

// CrossingResource is the representation of a crossing in the mint API.
type CrossingResource struct {
	ID          string `json:"id"`
	Created     int64  `json:"created"`
	Owner       string `json:"owner"`
	Propagation PgType `json:"propagation"`

	Offer  string   `json:"offer"`
	Amount *big.Int `json:"amount"`

	Status         TxStatus `json:"status"`
	Transaction    string   `json:"transaction"`
	TransactionHop int8     `json:"transaction_hop"`
}

// TransactionResource is the representation of a transaction in the mint API.
type TransactionResource struct {
	ID          string `json:"id"`
	Created     int64  `json:"created"`
	Owner       string `json:"owner"`
	Propagation PgType `json:"propagation"`

	Pair        string   `json:"pair"`
	Amount      *big.Int `json:"amount"`
	Destination string   `json:"destination"`
	Path        []string `json:"path"`

	Status TxStatus `json:"status"`
	Lock   string   `json:"lock"`
	Secret *string  `json:"secret"`

	Operations []OperationResource `json:"operations"`
	Crossings  []CrossingResource  `json:"crossings"`
}
