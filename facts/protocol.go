package facts

import "github.com/spolu/settl/model"

// SignatureResource reprensents a signature as returned by the API.
type SignatureResource struct {
	ID        string          `json:"id"`
	Created   int64           `json:"created"`
	Fact      string          `json:"fact"`
	PublicKey model.PublicKey `json:"public_key"`
	Signature string          `json"signature"`
}

// RevocationResource reprensents a revocation as returned by the API.
type RevocationResource struct {
	ID        string          `json:"id"`
	Created   int64           `json:"created"`
	Fact      string          `json:"fact"`
	PublicKey model.PublicKey `json:"public_key"`
	Signature string          `json"signature"`
}

// FactParams are the parameters used to create a fact.
type FactParams struct {
	Entity    model.Entity    `json:"entity"`
	Type      model.FctType   `json:"type"`
	Value     string          `json:"value"`
	PublicKey model.PublicKey `json:"public_key"`
	Signature string          `json:"signature"`
}

// FactResource represent the fact as returned by the API
type FactResource struct {
	ID          string               `json:"id"`
	Created     int64                `json:"created"`
	Entity      model.Entity         `json:"entity"`
	Type        model.FctType        `json:"type"`
	Value       string               `json:"value"`
	Signatures  []SignatureResource  `json:"signatures"`
	Revocations []RevocationResource `json:"revocation"`
}
