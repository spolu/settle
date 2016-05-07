package facts

import (
	"github.com/spolu/settl/model"
	"golang.org/x/net/context"
)

// Params

// FactParams are the parameters used to create a fact.
type FactParams struct {
	Account   model.PublicKey          `json:"public_key"`
	Type      model.FctType            `json:"type"`
	Value     string                   `json:"value"`
	Signature model.PublicKeySignature `json:"signature"`
}

// Resources

// AssertionResource represents an assertion as returned by the API.
type AssertionResource struct {
	ID        string                   `json:"id"`
	Created   int64                    `json:"created"`
	Fact      string                   `json:"fact"`
	Account   model.PublicKey          `json:"entity"`
	Signature model.PublicKeySignature `json:"signature"`
}

// NewAssertionResource renders a new AssertionResource from an assertion
// model.
func NewAssertionResource(
	ctx context.Context,
	assertion model.Assertion,
) *AssertionResource {
	return &AssertionResource{
		ID:        assertion.ID,
		Created:   assertion.Created,
		Fact:      assertion.Fact,
		Account:   assertion.Account,
		Signature: assertion.Signature,
	}
}

// RevocationResource reprensents a revocation as returned by the API.
type RevocationResource struct {
	ID        string                   `json:"id"`
	Created   int64                    `json:"created"`
	Fact      string                   `json:"fact"`
	Account   model.PublicKey          `json:"entity"`
	Signature model.PublicKeySignature `json:"signature"`
	Assertion *AssertionResource       `json:"assertion"`
}

// NewRevocationResource renders a new RevocationResource from a revocation
// model along with the assertion it revokes.
func NewRevocationResource(
	ctx context.Context,
	revocation model.Revocation,
	assertion model.Assertion,
) *RevocationResource {
	return &RevocationResource{
		ID:        revocation.ID,
		Created:   revocation.Created,
		Fact:      revocation.Fact,
		Account:   revocation.Account,
		Signature: revocation.Signature,
		Assertion: NewAssertionResource(ctx, assertion),
	}
}

// FactResource represents a fact as returned by the API.
type FactResource struct {
	ID          string               `json:"id"`
	Created     int64                `json:"created"`
	Account     model.PublicKey      `json:"entity"`
	Type        model.FctType        `json:"type"`
	Value       string               `json:"value"`
	Assertions  []AssertionResource  `json:"assertions"`
	Revocations []RevocationResource `json:"revocation"`
}

// NewFactResource renders a new FactResource from a fact object and its
// associated assertions and revocations ordered by created time.
func NewFactResource(
	ctx context.Context,
	fact model.Fact,
	assertions []model.Assertion,
	revocations []model.Revocation,
) (*FactResource, error) {
	return nil, nil
}
