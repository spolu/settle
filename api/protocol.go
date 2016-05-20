package api

// Params

// Resources

// TokenResource represents a token as returned by the API.
type TokenResource struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	ExpiresAt int64  `json:"expires_at"`
}
