package format

import (
	"bytes"
	"encoding/json"

	"github.com/spolu/settle/lib/errors"
)

// JSON formats an object into a JSON bytes.Buffer panicking if an error
// happens.
func JSON(
	response interface{},
) bytes.Buffer {
	var b bytes.Buffer
	formatted, err := json.Marshal(response)
	if err != nil {
		panic(errors.Trace(err))
	}
	json.HTMLEscape(&b, formatted)
	return b
}

// JSONIndented formats an object into an indented JSON bytes.Buffer panicking
// if an error happens.
func JSONIndented(
	response interface{},
) bytes.Buffer {
	var b bytes.Buffer
	formatted, err := json.Marshal(response)
	if err != nil {
		panic(errors.Trace(err))
	}
	json.Indent(&b, formatted, "", " ")
	return b
}

// JSONRaw formats an object into a json.RawMessage
func JSONRaw(
	response interface{},
) json.RawMessage {
	j := JSON(response)
	return json.RawMessage(j.Bytes())
}

// JSONPtr formats an object into a json.RawMessage pointer
func JSONPtr(
	response interface{},
) *json.RawMessage {
	j := JSON(response)
	msg := json.RawMessage(j.Bytes())
	return &msg
}

// JSONString formats an object into a string
func JSONString(
	response interface{},
) string {
	j := JSON(response)
	return string(j.Bytes())
}

// JSONIndentedString formats an object into a string
func JSONIndentedString(
	response interface{},
) string {
	j := JSONIndented(response)
	return string(j.Bytes())
}
