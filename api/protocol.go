package api

// Params

// UserParams are the parameters used to create a new user.
type UserParams struct {
	Username      string
	Address       string
	EncryptedSeed string
	Email         string
	Verifier      string
}

// OpType represent a native operation type.
type OpType string

const (
	// OpCreateAccount creates an account.
	OpCreateAccount OpType = "create_account"
	// OpPayment makes a direct payment.
	OpPayment OpType = "payment"
	// OpPathPayment makes a path payment.
	OpPathPayment OpType = "path_payment"
	// OpManageOffer manages an offer.
	OpManageOffer OpType = "manager_offer"
	// OpCreatePassiveOffer creates a passive offer.
	OpCreatePassiveOffer OpType = "create_passive_offer"
	// OpSetOptions sets options.
	OpSetOptions OpType = "set_options"
	// OpChangeTrust changes a trust line.
	OpChangeTrust OpType = "change_trust"
	// OpAllowTrust updates the authorized flag on an existing trustline.
	OpAllowTrust OpType = "allow_trust"
	// OpAccountMerge merges an account.
	OpAccountMerge OpType = "account_merge"
	// OpInflation runs the inflation.
	OpInflation OpType = "inflation"
	// OpManageData manages data for the account.
	OpManageData OpType = "manage_data"
)

var ParametersForOpType = map[protocol.OpType][]string{
	OpPayment:     []string{"destination", "asset", "amount"},
	OpPathPayment: []string{"send_asset", "send_max", "destination", "destination_asset", "destination_amount"},
}

// Resources

// ChallengeResource represents a challenge as returned by the API.
type ChallengeResource struct {
	Value   string `json:"value"`
	Created int64  `json:"created"`
}

// UserResource represents a challenge as returned by the API.
type UserResource struct {
	ID       string `json:"id"`
	Created  int64  `json:"created"`
	Livemode bool   `json:"livemode"`

	Username      string `json:"username"`
	Address       string `json:"address"`
	EncryptedSeed string `json:"encrypted_seed"`
	Email         string `json:"email,omitempty"`
	Verifier      string `json:"verifier,omitempty"`
}
