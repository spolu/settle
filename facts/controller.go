package facts

import (
	"fmt"
	"net/http"

	"goji.io/pat"

	"github.com/spolu/settl/model"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/format"
	"github.com/spolu/settl/util/respond"
	"github.com/spolu/settl/util/svc"
	"github.com/stellar/go-stellar-base/horizon"

	"golang.org/x/net/context"
)

type controller struct {
}

func (c *controller) loadLatestByAccountAndType(
	ctx context.Context,
	account model.PublicKey,
	t model.FctType,
) (*model.Fact, []model.Assertion, []model.Revocation, error) {
	fact, err := model.LoadLatestFactByAccountAndType(ctx, account, t)
	if err != nil {
		return nil, nil, nil, errors.Trace(err)
	} else if fact == nil {
		return nil, nil, nil, nil
	}

	assertions, err := model.LoadAssertionsByFact(ctx, fact.ID)
	if err != nil {
		return nil, nil, nil, errors.Trace(err)
	}
	revocations, err := model.LoadRevocationsByFact(ctx, fact.ID)
	if err != nil {
		return nil, nil, nil, errors.Trace(err)
	}

	return fact, assertions, revocations, nil
}

func (c *controller) CreateFact(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	factAccount := model.PublicKey(pat.Param(ctx, "account"))
	signAccount := model.PublicKey(r.PostFormValue("account"))

	if factAccount != signAccount {
		respond.Error(ctx, w, errors.Trace(
			errors.NewUserError(nil,
				400,
				"invalid_account",
				fmt.Sprintf("The account specified in the URL (%s), does not "+
					"match the account used to create and sign the fact (%s).",
					factAccount, signAccount),
			)))
		return
	}

	params := FactParams{
		Account:   factAccount,
		Type:      model.FctType(r.PostFormValue("type")),
		Value:     r.PostFormValue("value"),
		Signature: model.PublicKeySignature(r.PostFormValue("signature")),
	}

	_, err := horizon.DefaultPublicNetClient.LoadAccount(string(params.Account))
	if err != nil {
		respond.Error(ctx, w, errors.Trace(
			errors.NewUserError(err,
				400,
				"invalid_account",
				"The account provided could not be retrieved from Stellar. "+
					"You must create a Stellar account first to use this API.",
			)))
		return
	}

	fact, _, revocations, err := c.loadLatestByAccountAndType(
		ctx, params.Account, params.Type)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
	}

	if fact != nil {
		// Check that the fact has been revoked otherwise 400. A fact gets
		// revoked by a revocation from its owner, so there can be only one.
		revoked := false
		for _, r := range revocations {
			if r.Account == fact.Account {
				revoked = true
			}
		}
		if !revoked {
			respond.Error(ctx, w, errors.Trace(
				errors.NewUserError(nil,
					400,
					"fact_already_exists",
					fmt.Sprintf(
						"A fact with the same type already exists: %s. "+
							"You must revoke that fact first if you want to "+
							"create a new one with an updated value.", fact.ID,
					),
				)))
			return
		}
	}

	fact = model.NewFact(params.Account, params.Type, params.Value)
	assertion := model.NewAssertion(fact.ID, params.Account, params.Signature)

	if ok := assertion.Verify(fact); !ok {
		respond.Error(ctx, w, errors.Trace(
			errors.NewUserError(nil,
				400,
				"fact_verification_failed",
				"The fact signature failed to verify, "+
					"meaning that the signature sent is propably erroneous.",
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

	resource, err := NewFactResource(
		ctx, *fact, []model.Assertion{*assertion}, nil)
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err)) // 500
		return
	}

	respond.Created(ctx, w, svc.Resp{
		"fact": format.JSONPtr(*resource),
	})
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
