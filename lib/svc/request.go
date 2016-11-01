package svc

import (
	"encoding/json"

	"github.com/spolu/settle/lib/errors"
)

// Req is the structure used to make a request to a service
type Req map[string]*json.RawMessage

// Extract extracts a protocol from a request
func (h Req) Extract(
	protocol string,
	data interface{},
) error {
	raw, ok := h[protocol]
	if !ok || raw == nil {
		return errors.Trace(ErrProtocolExtraction{protocol})
	}
	if err := json.Unmarshal(*raw, data); err != nil {
		return errors.Trace(ErrProtocolExtraction{protocol})
	}
	return nil
}

// ExtractAll extracts a set of protocols given by the keys of `protocols` from
// `h`. Protocols are considered optional if they end with a '?' character. For
// example, the following code would read the protocol "foo" into the variable
// `a` and the protocol "bar" into the variable `b` if the request contains that
// protocol. If the "bar" protocol is not specified, `b` will retain its
// original value "optional".
//
//     a := "required"
//     b := "optional"
//     err := req.ExtractAll(map[string]interface{}{
//       "foo":  &a,
//       "bar?": &b,
//     })
//
// This function returns an error if we fail to extract any required protocol.
func (h Req) ExtractAll(
	protocols map[string]interface{},
) error {
	for protocol, data := range protocols {
		originalProtocol := protocol

		optional := protocol[len(protocol)-1] == '?'
		if optional {
			protocol = protocol[0 : len(protocol)-1]
		}

		// Extract the protocol. If this fails, revert to the original data if the
		// protocol was optional.
		err := h.Extract(protocol, data)
		if err != nil {
			if !optional {
				return err
			}
			data = protocols[originalProtocol]
		}
	}
	return nil
}
