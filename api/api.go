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
	mux.HandleFuncC(pat.Post("/users/:user/confirm"), c.controller.ConfirmUser)
	mux.HandleFuncC(pat.Post("/stellar/operations"), c.controller.CreateStellarOperation)
	mux.HandleFuncC(pat.Post("/stellar/operations/:operation/submit"), c.controller.SubmitStellarOperation)
}
