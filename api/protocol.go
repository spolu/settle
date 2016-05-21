package api

// Params

// UserParams are the parameters used to create a new user.
type UserParams struct {
	Username      string
	EncryptedSeed string
}

// Resources

// ChallengeResource represents a challenge as returned by the API.
type ChallengeResource struct {
	Value   string `json:"value"`
	Created int64  `json:"created"`
}

// UserResource represents a challenge as returned by the API.
type UserResource struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`

	Username      string `json:"username"`
	EncryptedSeed string `json:"encrypted_seed"`
}
