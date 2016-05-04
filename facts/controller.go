package facts

import (
	"net/http"

	"github.com/spolu/settl/model"

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
