package api

import "github.com/spolu/settl/model"

// Params

type UserParams struct {
	Username      string
	Address       model.Address
	EncryptedSeed string
}

// Resources

// TokenResource represents a token as returned by the API.
type TokenResource struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	ExpiresAt int64  `json:"expires_at"`
}
