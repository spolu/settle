package endpoint

import (
	"context"
	"net/http"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/respond"
	"github.com/spolu/settle/lib/svc"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

// EndPtName reprensents an endpoint name.
type EndPtName string

// registrar is used to register endpoints within the module.
var registrar = map[EndPtName](func(*http.Request) (Endpoint, error)){}

// Endpoint is the interface that endpoints need to implement.
type Endpoint interface {
	Validate(
		r *http.Request,
	) error

	Execute(
		ctx context.Context,
	) (*int, *svc.Resp, error)
}

// HandlerFor returns an handler for the given endpoint name.
func HandlerFor(
	name EndPtName,
) func(
	http.ResponseWriter,
	*http.Request,
) {
	return func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		ctx := r.Context()

		endpt, err := registrar[name](r)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err))
			return
		}

		err = endpt.Validate(r)
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err))
			return
		}

		status, resp, err := endpt.Execute(r.Context())
		if err != nil {
			respond.Error(ctx, w, errors.Trace(err))
			return
		}
		respond.Respond(ctx, w, *status, nil, *resp)
	}
}
