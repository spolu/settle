package facts

import (
	"net/http"

	"github.com/spolu/settl/model"
	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/respond"
	"github.com/stellar/go-stellar-base/keypair"

	"golang.org/x/net/context"
)

type controller struct {
}

func (c *controller) CreateFact(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {

	params := FactParams{
		Type:      model.FctType(r.PostFormValue("type")),
		Value:     r.PostFormValue("value"),
		Account:   model.PublicKey(r.PostFormValue("account")),
		Signature: model.PublicKeySignature(r.PostFormValue("signature")),
	}

	from, err := keypair.Parse(string(params.Account))
	if err != nil {
		respond.Error(ctx, w, errors.Trace(err))
		return
	}
	_ = from

	fact := model.NewFact(params.Account, params.Type, params.Value)
	assertion := model.NewAssertion(fact.ID, params.Account, params.Signature)

	if ok := assertion.Verify(fact); !ok {
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
