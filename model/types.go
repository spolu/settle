package model

// PublicKey represents a Stellar public key.
type PublicKey string

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
