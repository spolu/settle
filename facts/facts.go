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
	mux.HandleFuncC(pat.Post("/accounts/:account/facts"), c.controller.CreateFact)
	mux.HandleFuncC(pat.Get("/accounts/:account/facts/:fact"), c.controller.RetrieveFact)
	mux.HandleFuncC(pat.Post("/accounts/:account/facts/:fact/assertions"), c.controller.CreateAssertion)
	mux.HandleFuncC(pat.Post("/accounts/:account/facts/:fact/revocations"), c.controller.CreateRevocation)
}
