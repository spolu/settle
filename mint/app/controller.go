package app

import (
	"github.com/spolu/settle/mint/endpoint"
	"goji.io"
	"goji.io/pat"
)

// Controller binds the API
type Controller struct{}

// Bind registers the API routes.
func (c *Controller) Bind(
	mux *goji.Mux,
) {
	// Local.
	mux.HandleFunc(pat.Post("/assets"), endpoint.HandlerFor(endpoint.EndPtCreateAsset))
	// mux.HandleFunc(pat.Get("/assets/:asset/operations"), c.controller.RetrieveOperations)
	mux.HandleFunc(pat.Post("/assets/:asset/operations"), endpoint.HandlerFor(endpoint.EndPtCreateOperation))
	// mux.HandleFunc(pat.Get("/assets/:asset/operations/:operation"), c.controller.RetrieveOperation)
	// mux.HandleFunc(pat.Get("/assets/:asset/balances/:address"), c.controller.RetrieveBalance)

	// Public.
	mux.HandleFunc(pat.Get("/offers/:offer"), endpoint.HandlerFor(endpoint.EndPtRetrieveOffer))
	mux.HandleFunc(pat.Post("/offers"), endpoint.HandlerFor(endpoint.EndPtCreateOffer))
	//mux.HandleFunc(pat.Post("/assets/offers/:offer/close"), c.controller.CloseOffer)

	//mux.HandleFunc(pat.Post("/assets/:asset/transactions"), c.controller.CreateTransaction)
	//mux.HandleFunc(pat.Post("/assets/:asset/transactions/:transaction/settle"), c.controller.SettleOperation)
}
