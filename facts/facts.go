package facts

import (
	"goji.io"
	"goji.io/pat"
)

// Configuration is used to create and bind the APi controller
type Configuration struct {
	controller *controller
}

// Init initializes the API controller
func (c *Configuration) Init() error {
	return nil
}

// Bind registers the API routes
func (c *Configuration) Bind(
	mux *goji.Mux,
) {
	mux.HandleFuncC(pat.Post("/facts"), c.controller.CreateFact)
	mux.HandleFuncC(pat.Post("/facts/:fact_id/signatures"), c.controller.CreateSignature)
	mux.HandleFuncC(pat.Post("/facts/:fact_id/revocations"), c.controller.CreateRevocation)
}
