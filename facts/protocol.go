package facts

import (
	"github.com/spolu/settl/model"
	"github.com/spolu/settl/util/errors"
	"golang.org/x/net/context"
)

// Params

// FactParams are the parameters used to create a fact.
type FactParams struct {
	Account   model.PublicKey
	Type      model.FctType
	Value     string
	Signature model.PublicKeySignature
}

// Resources

// AssertionResource represents an assertion as returned by the API.
type AssertionResource struct {
	ID        string                   `json:"id"`
	Created   int64                    `json:"created"`
	Fact      string                   `json:"fact"`
	Account   model.PublicKey          `json:"account"`
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
	Account   model.PublicKey          `json:"account"`
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
	Account     model.PublicKey      `json:"account"`
	Type        model.FctType        `json:"type"`
	Value       string               `json:"value"`
	Assertions  []AssertionResource  `json:"assertions"`
	Revocations []RevocationResource `json:"revocations,omitempty"`
}

// NewFactResource renders a new FactResource from a fact object and its
// associated assertions and revocations assumed to be ordered (descending) by
// created time. If they are not ordered, the result is undefined.
func NewFactResource(
	ctx context.Context,
	fact model.Fact,
	assertions []model.Assertion,
	revocations []model.Revocation,
) (*FactResource, error) {
	var revRes []RevocationResource
	var assRes []AssertionResource
	for _, r := range revocations {
		var v *model.Assertion
		var idx = -1
		for i, a := range assertions {
			if a.Account == r.Account && a.Created < r.Created {
				v = &a
				idx = i
				break
			}
			if v != nil {
				assertions = append(assertions[:idx], assertions[idx+1:]...)
				revRes = append(revRes, *NewRevocationResource(ctx, r, *v))
			} else {
				return nil, errors.Newf("No assertion for revocation: %s", r.ID)
			}
		}
	}
	for _, a := range assertions {
		assRes = append(assRes, *NewAssertionResource(ctx, a))
	}

	return &FactResource{
		ID:          fact.ID,
		Created:     fact.Created,
		Account:     fact.Account,
		Type:        fact.Type,
		Value:       fact.Value,
		Assertions:  assRes,
		Revocations: revRes,
	}, nil
}
