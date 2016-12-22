package register

const (
	// Version is the current protocol version.
	Version string = "0"
	// TimeResolutionNs is the resolution of our time variables in nanoseconds
	// (aka resolution in milliseconds).
	TimeResolutionNs int64 = 1000 * 1000
)

// UsrStatus is the status of a user.
type UsrStatus string

const (
	// UsrStUnverified is an unverified user.
	UsrStUnverified UsrStatus = "unverified"
	// UsrStVerified is a verified user
	UsrStVerified UsrStatus = "verified"
)

// CredentialsResource is the representation of user credentials.
type CredentialsResource struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

// UserResource is the representation of a user in the register API.
type UserResource struct {
	ID      string    `json:"id"`
	Created int64     `json:"created"`
	Status  UsrStatus `json:"status"`

	Username string `json:"username"`
	Email    string `json:"email"`

	Credentials *CredentialsResource `json:"credentials"`
}
