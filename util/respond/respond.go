package respond

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spolu/settl/util/errors"
	"github.com/spolu/settl/util/format"
	"github.com/spolu/settl/util/logging"
	"github.com/spolu/settl/util/svc"
	"golang.org/x/net/context"
)

func panicError() errors.ConcreteUserError {
	return errors.ConcreteUserError{
		Type: errors.InternalError,
		Code: "api_error",
		Message: "Sorry! There was an error while processing your request. " +
			"We have already been notified of the problem, but please contact " +
			"support@stripe.com with any questions you may have.",
	}
}

// userStatusCode maps from UserError ErrorType to HTTP status code.
func userStatusCode(
	err errors.UserError,
) int {
	switch err.Type() {
	case errors.InvalidRequest:
		return http.StatusBadRequest
	case errors.ActionFailed:
		return http.StatusPaymentRequired
	case errors.NotFound:
		return http.StatusNotFound
	case errors.InternalError:
		return http.StatusInternalServerError
	}
	return http.StatusInternalServerError
}

// errorResponse generates an res.Resp object based on a ConcreteUserError,
// injecting data from the context in the process.
func errorResponse(
	ctx context.Context,
	err errors.ConcreteUserError,
) svc.Resp {
	resp := svc.Resp{
		"error": format.JSONPtr(err),
	}

	injectContext(ctx, &resp)
	return resp
}

// Success is used to successfully respond with status 200. Error cases should
// be hanlded by Recoverer (through panics)
func Success(
	ctx context.Context,
	w http.ResponseWriter,
	resp svc.Resp,
) {
	injectContext(ctx, &resp)
	Respond(ctx, w, http.StatusOK, nil, resp)
}

// Error triage the error and respond with its content if it's a UserError,
// otherwise responds with a default `api_error`
func Error(
	ctx context.Context,
	w http.ResponseWriter,
	err error,
) {
	// Handle UserError
	if errors.IsUserError(err) {
		if e, ok := err.(errors.UserError); ok {
			logging.Logf(ctx,
				"Responding with UserError: code=%q message=%q",
				e.Code(), e.Message())
			b := errors.Build(e)
			for _, line := range errors.ErrorStack(err) {
				logging.Logf(ctx, "  %s", line)
			}

			resp := errorResponse(ctx, b)
			Respond(ctx, w, userStatusCode(e), nil, resp)
		} else {
			panic(fmt.Errorf("Unexpected non-UserError"))
		}
	} else {
		logging.Logf(ctx,
			"Responding with non-UserError: error=%q", err.Error())
		body := panicError()
		for _, line := range errors.ErrorStack(err) {
			logging.Logf(ctx, "  %s", line)
		}
		errors.Sentry(ctx, err)

		resp := errorResponse(ctx, body)
		Respond(ctx, w, http.StatusInternalServerError, nil, resp)
	}
}

// Respond is used to generate a response manually setting the status code,
// headers and body
func Respond(
	ctx context.Context,
	w http.ResponseWriter,
	status int,
	header http.Header,
	data interface{},
) {
	w.Header().Add("Content-Type", "application/json")
	for header, values := range header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}

	if status != 0 {
		w.WriteHeader(status)
	}

	if data != nil {
		if err := formatJSON(data, w); err != nil {
			logging.Logf(ctx, "Failed to write body")
		}
	}
}

func formatJSON(
	response interface{},
	w io.Writer,
) error {
	var b bytes.Buffer
	formatted, err := json.Marshal(response)
	if err != nil {
		return err
	}
	json.HTMLEscape(&b, formatted)
	if _, err := b.Write([]byte("\n")); err != nil {
		return err
	}
	if _, err := io.Copy(w, &b); err != nil {
		return err
	}
	return nil
}
