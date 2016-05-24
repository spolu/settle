package api

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

// Bind registers the API routes
func (c *Configuration) Bind(
	mux *goji.Mux,
) {
	mux.HandleFuncC(pat.Get("/challenges"), c.controller.RetrieveChallenges)

	mux.HandleFuncC(pat.Post("/users"), c.controller.CreateUser)

	mux.HandleFuncC(pat.Post("/native/operations"), c.controller.CreateNativeOperation)
	mux.HandleFuncC(pat.Post("/native/operations/:operation/submit"), c.controller.SubmitNativeOperation)
}
