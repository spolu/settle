package errors

// ErrorType is an enumeration of acceptable error types that can be exposed to
// external clients of the system. Error types informs the nature of the error
// and whether that error is due to the request or an internal error.
type ErrorType string

const (
	// NonUserError unset ErroType
	NonUserError ErrorType = ""
	// InvalidRequest the request is missing a paramter or invalid
	InvalidRequest ErrorType = "invalid_request"
	// ActionFailed the request is valid but failed
	ActionFailed ErrorType = "action_failed"
	// NotFound the path/resource requested is not found
	NotFound ErrorType = "not_found"
	// InternalError something went wrong internally
	InternalError ErrorType = "internal_error"
)

// UserError is the interface an error has to comply to to be consumable by an
// external client.
type UserError interface {
	Type() ErrorType
	Code() string
	Message() string
	Metadata() map[string]string
	Cause() error
}

// ConcreteUserError is the materialization of the UserError for marshalling
type ConcreteUserError struct {
	Type     ErrorType         `json:"type"`
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Build constructs a ConcreteUserError from a UserError. It also assigns a
// unique ID to the ConcreteUserError for log/reply correlation.
func Build(err UserError) ConcreteUserError {
	return ConcreteUserError{
		Type:     err.Type(),
		Code:     err.Code(),
		Message:  err.Message(),
		Metadata: err.Metadata(),
	}
}
