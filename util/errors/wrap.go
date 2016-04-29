package errors

import (
	"runtime"
)

// wrap is the internal error type used to attach information to an underlying
// error. As information is attached, the error is wrapped in wrap structures
// each containing details about the error.
type wrap struct {
	traceFile string
	traceLine int

	traceMessage string

	errType     ErrorType
	errCode     string
	errMessage  string
	errMetadata map[string]string

	previous error
}

// Type returns the most recent consumable ErrorType attached to the error
func (e *wrap) Type() ErrorType {
	if e.errType != "" {
		return e.errType
	}
	switch e := e.previous.(type) {
	case *wrap:
		return e.Type()
	case UserError:
		return e.Type()
	default:
		return NonUserError
	}
}

// Code returns the most recent consumable error code attached to the error
func (e *wrap) Code() string {
	if e.errCode != "" {
		return e.errCode
	}
	switch e := e.previous.(type) {
	case *wrap:
		return e.Code()
	case UserError:
		return e.Code()
	default:
		return ""
	}
}

// Message returns the most recent consumable message attached to the error
func (e *wrap) Message() string {
	if e.errMessage != "" {
		return e.errMessage
	}
	switch e := e.previous.(type) {
	case *wrap:
		return e.Message()
	case UserError:
		return e.Message()
	default:
		return ""
	}
}

// Metadata returns the most recent consumable metadata attached to the error
func (e *wrap) Metadata() map[string]string {
	if e.errMetadata != nil {
		return e.errMetadata
	}
	switch e := e.previous.(type) {
	case *wrap:
		return e.Metadata()
	case UserError:
		return e.Metadata()
	default:
		return nil
	}
}

// Cause returns the underlying error if not nil
func (e *wrap) Cause() error {
	switch e := e.previous.(type) {
	case *wrap:
		return e.Cause()
	case UserError:
		return e.Cause()
	default:
		return e
	}
}

// Error returns the error message of the underlying error if not nil otherwise
// it returns the error stack consumable message.
func (e *wrap) Error() string {
	err := e.Cause()
	if err != nil {
		return err.Error()
	}
	return e.Message()
}

// StackTrace returns the full stack of information attached to the error
func (e *wrap) StackTrace() []string {
	return ErrorStack(e)
}

func (e *wrap) setLocation(callDepth int) {
	_, file, line, _ := runtime.Caller(callDepth + 1)
	e.traceFile = file
	e.traceLine = line
}

func (e *wrap) location() (filename string, line int) {
	return e.traceFile, e.traceLine
}
