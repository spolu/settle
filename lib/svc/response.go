package svc

import (
	"encoding/json"

	"github.com/spolu/settle/lib/errors"
)

// Resp is the structure used to respond to a request
type Resp map[string]*json.RawMessage

// Extract extracts a protocol from a response
func (h Resp) Extract(
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
