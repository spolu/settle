package app

import (
	"github.com/spolu/settle/register/endpoint"
	"goji.io"
	"goji.io/pat"
)

// Controller binds the API
type Controller struct{}

// Bind registers the API routes.
func (c *Controller) Bind(
	mux *goji.Mux,
) {
	// Public.
	mux.HandleFunc(pat.Post("/users"), endpoint.HandlerFor(endpoint.EndPtCreateUser))
	mux.HandleFunc(pat.Get("/users/:username"), endpoint.HandlerFor(endpoint.EndPtRetrieveUser))
	mux.HandleFunc(pat.Post("/users/:username/roll"), endpoint.HandlerFor(endpoint.EndPtRollUser))

}
