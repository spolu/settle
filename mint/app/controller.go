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
	// Authenticated.
	mux.HandleFunc(pat.Post("/assets"), endpoint.HandlerFor(endpoint.EndPtCreateAsset))
	mux.HandleFunc(pat.Post("/offers"), endpoint.HandlerFor(endpoint.EndPtCreateOffer))
	mux.HandleFunc(pat.Post("/transactions"), endpoint.HandlerFor(endpoint.EndPtCreateTransaction))
	mux.HandleFunc(pat.Post("/offers/:offer/close"), endpoint.HandlerFor(endpoint.EndPtCloseOffer))

	mux.HandleFunc(pat.Get("/assets"), endpoint.HandlerFor(endpoint.EndPtListAssets))
	mux.HandleFunc(pat.Get("/balances"), endpoint.HandlerFor(endpoint.EndPtListBalances))
	mux.HandleFunc(pat.Get("/assets/:asset/balances"), endpoint.HandlerFor(endpoint.EndPtListAssetBalances))
	// mux.HandleFunc(pat.Get("/assets/:asset/operations"), endpoint.HandlerFor(endpoint.EndPtListOperations))

	// Mixed.
	mux.HandleFunc(pat.Post("/transactions/:transaction/settle"), endpoint.HandlerFor(endpoint.EndPtSettleTransaction))
	mux.HandleFunc(pat.Post("/transactions/:transaction/cancel"), endpoint.HandlerFor(endpoint.EndPtCancelTransaction))

	// Public.
	mux.HandleFunc(pat.Get("/offers/:offer"), endpoint.HandlerFor(endpoint.EndPtRetrieveOffer))
	mux.HandleFunc(pat.Get("/operations/:operation"), endpoint.HandlerFor(endpoint.EndPtRetrieveOperation))
	mux.HandleFunc(pat.Get("/transactions/:transaction"), endpoint.HandlerFor(endpoint.EndPtRetrieveTransaction))
	mux.HandleFunc(pat.Get("/balances/:balance"), endpoint.HandlerFor(endpoint.EndPtRetrieveBalance))

	mux.HandleFunc(pat.Post("/transactions/:transaction"), endpoint.HandlerFor(endpoint.EndPtCreateTransaction))
	mux.HandleFunc(pat.Post("/operations/:operation"), endpoint.HandlerFor(endpoint.EndPtPropagateOperation))
	mux.HandleFunc(pat.Post("/offers/:offer"), endpoint.HandlerFor(endpoint.EndPtPropagateOffer))
	mux.HandleFunc(pat.Post("/balances/:balance"), endpoint.HandlerFor(endpoint.EndPtPropagateBalance))

	mux.HandleFunc(pat.Get("/assets/:asset"), endpoint.HandlerFor(endpoint.EndPtRetrieveAsset))
	mux.HandleFunc(pat.Get("/assets/:asset/offers"), endpoint.HandlerFor(endpoint.EndPtListAssetOffers))
}
