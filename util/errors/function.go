package errors

import (
	"fmt"
	"strings"
)

// Newf creates a new raw error and trace it.
func Newf(format string, args ...interface{}) error {
	err := &wrap{
		previous: fmt.Errorf(format, args...),
	}
	err.setLocation(1)
	return err
}

// Trace attach a location to the error. It should be called each time an error
// is returned. If the error is nil, it returns nil.
func Trace(other error) error {
	if other == nil {
		return nil
	}
	err := &wrap{
		previous: other,
	}
	err.setLocation(1)
	return err
}

// Tracef attach a location and an annotation to the error. If the error is nil
// it returns nil.
func Tracef(other error, format string, args ...interface{}) error {
	if other == nil {
		return nil
	}
	err := &wrap{
		traceMessage: fmt.Sprintf(format, args...),
		previous:     other,
	}
	err.setLocation(1)
	return err
}

// Meta attaches a metadata object to the error. If the error is nil, it
// returns nil
func Meta(other error, meta map[string]string) error {
	if other == nil {
		return nil
	}
	err := &wrap{
		previous:    other,
		errMetadata: meta,
	}
	err.setLocation(1)
	return err
}

// InvalidRequestf mark this error as an UserError of type InvalidRequest and
// attach a consumable error code and message. If the error is nil, it returns
// an error.
func InvalidRequestf(other error, code string, format string, args ...interface{}) error {
	err := &wrap{
		errType:    InvalidRequest,
		errCode:    code,
		errMessage: fmt.Sprintf(format, args...),
		previous:   other,
	}
	err.setLocation(1)
	return err
}

// ActionFailedf mark this error as an UserError of type ActionFailed and
// attach a consumable error code and message. If the error is nil, it returns
// an error.
func ActionFailedf(other error, code string, format string, args ...interface{}) error {
	err := &wrap{
		errType:    ActionFailed,
		errCode:    code,
		errMessage: fmt.Sprintf(format, args...),
		previous:   other,
	}
	err.setLocation(1)
	return err
}

// NotFoundf mark this error as an UserError of type NotFound and attach a
// consumable error code and message. If the error is nil, it returns an error.
func NotFoundf(other error, code string, format string, args ...interface{}) error {
	err := &wrap{
		errType:    NotFound,
		errCode:    code,
		errMessage: fmt.Sprintf(format, args...),
		previous:   other,
	}
	err.setLocation(1)
	return err
}

// InternalErrorf mark this error as an UserError of type InternalError and
// attach a consumable error code and message. If the error is nil, it returns
// an error.
func InternalErrorf(other error, code string, format string, args ...interface{}) error {
	err := &wrap{
		errType:    InternalError,
		errCode:    code,
		errMessage: fmt.Sprintf(format, args...),
		previous:   other,
	}
	err.setLocation(1)
	return err
}

// IsUserError returns true if the error was marked as a UserError
func IsUserError(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(UserError); ok {
		if e.Type() != NonUserError {
			return true
		}
	}
	return false
}

// Type extracts the most recent consumable ErrorType attached to the error if
// any.
func Type(err error) ErrorType {
	if e, ok := err.(UserError); ok {
		return e.Type()
	}
	return NonUserError
}

// Code extracts the most recent consumable message attached to the error if
// any.
func Code(err error) string {
	if e, ok := err.(UserError); ok {
		return e.Code()
	}
	return ""
}

// Message extracts the most recent consumable message attached to the error if
// any.
func Message(err error) string {
	if e, ok := err.(UserError); ok {
		return e.Message()
	}
	return ""
}

// Metadata returns the most recent consumable metadata attached to the error
// if any
func Metadata(err error) map[string]string {
	if e, ok := err.(UserError); ok {
		return e.Metadata()
	}
	return nil
}

// Cause returns the underlying raw error if not nil.
func Cause(err error) (ret error) {
	if err, ok := err.(*wrap); ok {
		ret = err.Cause()
		return
	}
	ret = err
	return
}

// ErrorStack returns the full stack of information attached to this error.
func ErrorStack(err error) []string {
	if err == nil {
		return []string{}
	}

	var lines []string
	for {
		var buff []byte
		if e, ok := err.(*wrap); ok {
			file, line := e.location()
			if file != "" {
				buff = append(buff, " "...)
				buff = append(buff, fmt.Sprintf("%s:%d", file, line)...)
				buff = append(buff, ": "...)
			}

			if e.errType != "" && e.traceMessage != "" {
				panic(fmt.Errorf("Mixed errType and traceMessage"))
			}

			if e.errType != "" {
				buff = append(buff, fmt.Sprintf("[%s] {%s} %s", e.errType, e.errCode, e.errMessage)...)
			} else {
				buff = append(buff, fmt.Sprintf("[trace] %s", e.traceMessage)...)
			}

			err = e.previous
		} else {
			buff = append(buff, err.Error()...)
			err = nil
		}
		lines = append(lines, string(buff))
		if err == nil {
			break
		}
	}

	// reverse the lines to get the original error, which was at the end of
	// the list, back to the start.
	var result []string
	for i := len(lines); i > 0; i-- {
		result = append(result, lines[i-1])
	}
	return result
}

// Details returned a formatted ErrorStack string
func Details(err error) string {
	return strings.Join(ErrorStack(err), "\n")
}
