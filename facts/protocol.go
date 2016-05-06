package facts

import (
	"github.com/spolu/settl/model"
	"golang.org/x/net/context"
)

// Resources

// AssertionResource represents an assertion as returned by the API.
type AssertionResource struct {
	ID        string                   `json:"id"`
	Created   int64                    `json:"created"`
	Fact      string                   `json:"fact"`
	Account   model.PublicKey          `json:"entity"`
	Signature model.PublicKeySignature `json:"signature"`
}

// NewFactResource renders a new FactResource from a fact object and its
// associated assertions and revocations ordered by created time.
func NewFactResource(
	ctx context.Context,
	assertions *Assertion,
	revocations []*Revocation,
) (*FactResource, error) {
}

// RevocationResource reprensents a revocation as returned by the API.
type RevocationResource struct {
	ID        string                   `json:"id"`
	Created   int64                    `json:"created"`
	Fact      string                   `json:"fact"`
	Account   model.PublicKey          `json:"entity"`
	Signature model.PublicKeySignature `json:"signature"`
	Assertion AssertionResource        `json:"assertion"`
}

// FactResource represents a fact as returned by the API.
type FactResource struct {
	ID          string               `json:"id"`
	Created     int64                `json:"created"`
	Account     model.PublicKey      `json:"entity"`
	Type        model.FctType        `json:"type"`
	Value       string               `json:"value"`
	Signatures  []SignatureResource  `json:"signatures"`
	Revocations []RevocationResource `json:"revocation"`
}

// NewFactResource renders a new FactResource from a fact object and its
// associated assertions and revocations ordered by created time.
func NewFactResource(
	ctx context.Context,
	fact *Fact,
	assertions []*Assertion,
	revocations []*Revocation,
) (*FactResource, error) {
}

// Params

// FactParams are the parameters used to create a fact.
type FactParams struct {
	Account   model.PublicKey          `json:"public_key"`
	Type      model.FctType            `json:"type"`
	Value     string                   `json:"value"`
	Signature model.PublicKeySignature `json:"signature"`
}
