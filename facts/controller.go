package facts

import (
	"net/http"

	"github.com/spolu/settl/model"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/respond"
	"github.com/stellar/go-stellar-base/horizon"

	"golang.org/x/net/context"
)

type controller struct {
}

func (c *controller) loadFullFactByAccountAndType(
	ctx context.Context,
	account PublicKey,
	t FctType,
) (*Fact, []*Assertion, []*Revocation, error) {
	fact, err := model.LoadLatestFactByAccountAndType(
		ctx,
		params.Account,
		params.Type,
	)
	if err != nil {
		return nil, nil, nil, errors.Trace(err)
	} else if fact == nil {
		return nil, nil, nil, nil
	}

	assertions := model.LoadAssertionsByFact(
		ctx,
		fact.ID,
	)
	revocations := model.LoadRevocationsByFact(
		ctx,
		fact.ID,
	)
}

func (c *controller) CreateFact(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	params := FactParams{
		Account:   model.PublicKey(r.PostFormValue("account")),
		Type:      model.FctType(r.PostFormValue("type")),
		Value:     r.PostFormValue("value"),
		Signature: model.PublicKeySignature(r.PostFormValue("signature")),
	}

	_, err := horizon.DefaultPublicNetClient.LoadAccount(string(params.Account))
	if err != nil {
		respond.Error(ctx, w, errors.Trace(
			errors.NewUserError(err,
				400,
				"invalid_stellar_account",
				"The account provided could not be retrieved from Stellar.",
			)))
		return
	}

	fact = model.NewFact(params.Account, params.Type, params.Value)
	assertion := model.NewAssertion(fact.ID, params.Account, params.Signature)

	if ok := assertion.Verify(fact); !ok {
		respond.Error(ctx, w, errors.Trace(
			errors.NewUserError(nil,
				400,
				"fact_verification_failed",
				"The fact signature failed to verify.",
			)))
		return
	}

	err = error(nil)
	err = assertion.Save(ctx)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}
	err = fact.Save(ctx)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

}

func (c *controller) CreateSignature(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}

func (c *controller) CreateRevocation(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
}
