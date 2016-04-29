package facts

// PublicKey represents a Stellar public key.
type PublicKey string

// Entity represents an entity as its public key.
type Entity PublicKey

// Signature reprensents a signature as returned by the API.
type Signature struct {
	ID        string    `json:"id"`
	Created   int64     `json:"created"`
	Fact      string    `json:"fact"`
	PublicKey PublicKey `json:"public_key"`
	Signature string    `json"signature"`
}

// Revocation reprensents a revocation as returned by the API.
type Revocation struct {
	ID        string    `json:"id"`
	Created   int64     `json:"created"`
	Fact      string    `json:"fact"`
	PublicKey PublicKey `json:"public_key"`
	Signature string    `json"signature"`
}

// FctType are the possible types for a Settl fact.
type FctType string

const (
	// FctName full name of an individual or organization.
	FctName FctType = "name"
	// FctDateOfBirth date of birth or an individual.
	FctDateOfBirth FctType = "date_of_birth"
	// FctDateOfCreation date of creation of an organization.
	FctDateOfCreation FctType = "date_of_creation"
	// FctEmail fully qualified email address.
	FctEmail FctType = "email"
	// FctURL fully qualified URL.
	FctURL FctType = "url"
	// FctPhone fully qualified phone number.
	FctPhone FctType = "phone"
	// FctTwitter Twitter handle without preceeding `@`.
	FctTwitter FctType = "twitter"
	// FctGithub Github handle.
	FctGithub FctType = "github"
	// FctBankAccountHash hash of a bank account using the entity as nonce.
	FctBankAccountHash FctType = "bank_account_hash"
	// FctTaxIDHash hash of the TaxID (SSN,...) using the entity as nonce.
	FctTaxIDHash FctType = "tax_id_hash"
)

// FactParams are the parameters used to create a fact.
type FactParams struct {
	Entity    Entity    `json:"entity"`
	Type      FctType   `json:"type"`
	Value     string    `json:"value"`
	PublicKey PublicKey `json:"public_key"`
	Signature string    `json:"signature"`
}

// Fact represent the fact as returned by the API
type Fact struct {
	ID          string       `json:"id"`
	Created     int64        `json:"created"`
	Entity      Entity       `json:"entity"`
	Type        FctType      `json:"type"`
	Value       string       `json:"value"`
	Signatures  []Signature  `json:"signatures"`
	Revocations []Revocation `json:"revocation"`
}
