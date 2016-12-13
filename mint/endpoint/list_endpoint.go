package endpoint

import (
	"net/http"
	"time"

	"github.com/spolu/settle/lib/errors"
)

// ListEndpoint is an helper object to implement list endpoints.
type ListEndpoint struct {
	CreatedBefore time.Time
	Limit         uint
}

// Validate validates the input parameters.
func (e *ListEndpoint) Validate(
	r *http.Request,
) error {
	ctx := r.Context()

	// Validate limit.
	limit, err := ValidateLimit(ctx, r.URL.Query().Get("limit"))
	if err != nil {
		return errors.Trace(err)
	}
	e.Limit = *limit

	// Validate created_before.
	createdBefore, err := ValidateCreatedBefore(ctx,
		r.URL.Query().Get("created_before"))
	if err != nil {
		return errors.Trace(err)
	}
	e.CreatedBefore = *createdBefore

	return nil
}
