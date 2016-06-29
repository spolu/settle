package mint

import (
	"goji.io"
	"goji.io/pat"
)

const (
	// Version is the current version.
	Version string = "0.0.1"
)

// Configuration is used to create and bind the APi controller
type Configuration struct {
	MintHost   string
	controller *controller
}

// Init initializes the API controller
func (c *Configuration) Init() error {
	c.controller = &controller{
		mintHost: c.MintHost,
		client:   &Client{},
	}
	return nil
}

// Bind registers the API routes.
func (c *Configuration) Bind(
	mux *goji.Mux,
) {
	mux.HandleFuncC(pat.Post("/assets"), c.controller.CreateAsset)
	mux.HandleFuncC(pat.Post("/assets/:asset/issue"), c.controller.IssueAsset)

	//mux.HandleFuncC(pat.Post("/assets/:asset/operations"), c.controller.CreateOperation)

	//mux.HandleFuncC(pat.Post("/assets/:asset/offers"), c.controller.CreateOffer)
	//mux.HandleFuncC(pat.Post("/assets/:asset/offers/:offer/close"), c.controller.CloseOffer)

	//mux.HandleFuncC(pat.Post("/assets/:asset/transactions"), c.controller.CreateTransaction)
	//mux.HandleFuncC(pat.Post("/assets/:asset/transactions/:transaction/settle"), c.controller.SettleOperation)
}
