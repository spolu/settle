package mint

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
	c.controller = &controller{}
	return nil
}

// Bind registers the API routes.
func (c *Configuration) Bind(
	mux *goji.Mux,
) {
	mux.HandleFuncC(pat.Post("/assets/:asset/offers"), c.controller.CreateOffer)
	mux.HandleFuncC(pat.Post("/assets/:asset/offers/:offer/close"), c.controller.CloseOffer)
	mux.HandleFuncC(pat.Post("/assets/:asset/transactions"), c.controller.CreateTransaction)
	mux.HandleFuncC(pat.Post("/assets/:asset/transactions/:transaction/settle"), c.controller.SettleOperation)
}
