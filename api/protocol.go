package api

// Params

// UserParams are the parameters used to create a new user.
type UserParams struct {
	Username      string
	Address       string
	EncryptedSeed string
	Email         string
	Verifier      string
}

// Resources

// ChallengeResource represents a challenge as returned by the API.
type ChallengeResource struct {
	Value   string `json:"value"`
	Created int64  `json:"created"`
}

// UserResource represents a challenge as returned by the API.
type UserResource struct {
	ID       string `json:"id"`
	Created  int64  `json:"created"`
	Livemode bool   `json:"livemode"`

	Username      string `json:"username"`
	Address       string `json:"address"`
	EncryptedSeed string `json:"encrypted_seed"`
	Email         string `json:"email,omitempty"`
	Verifier      string `json:"verifier,omitempty"`
}
