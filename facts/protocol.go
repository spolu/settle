package facts

import "github.com/spolu/settl/model"

// Resources

// SignatureResource represents a signature as returned by the API.
type SignatureResource struct {
	ID        string                   `json:"id"`
	Created   int64                    `json:"created"`
	Fact      string                   `json:"fact"`
	Account   model.PublicKey          `json:"entity"`
	Signature model.PublicKeySignature `json:"signature"`
}

// RevocationResource reprensents a revocation as returned by the API.
type RevocationResource struct {
	ID        string                   `json:"id"`
	Created   int64                    `json:"created"`
	Fact      string                   `json:"fact"`
	Account   model.PublicKey          `json:"entity"`
	Signature model.PublicKeySignature `json:"signature"`
}

// FactResource represent the fact as returned by the API
type FactResource struct {
	ID          string               `json:"id"`
	Created     int64                `json:"created"`
	Account     model.PublicKey      `json:"entity"`
	Type        model.FctType        `json:"type"`
	Value       string               `json:"value"`
	Signatures  []SignatureResource  `json:"signatures"`
	Revocations []RevocationResource `json:"revocation"`
}

// Params

// FactParams are the parameters used to create a fact.
type FactParams struct {
	Type      model.FctType            `json:"type"`
	Value     string                   `json:"value"`
	Account   model.PublicKey          `json:"public_key"`
	Signature model.PublicKeySignature `json:"signature"`
}
