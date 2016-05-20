package api

import "github.com/spolu/settl/model"

// Params

// UserParams are the parameters used to create a new user.
type UserParams struct {
	Username      string
	Address       model.Address
	EncryptedSeed string
}

// Resources

// ChallengeResource represents a challenge as returned by the API.
type ChallengeResource struct {
	Value   string `json:"value"`
	Created int64  `json:"created"`
}
