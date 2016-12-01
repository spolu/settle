package respond

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/format"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/lib/svc"
)

func panicError() errors.UserError {
	return errors.NewUserError(nil,
		http.StatusInternalServerError,
		"internal_error",
		"There was an error while processing your request.",
	)
}

// errorResponse generates an res.Resp object based on a ConcreteUserError,
// injecting data from the context in the process.
func errorResponse(
	ctx context.Context,
	err *errors.ConcreteUserError,
) svc.Resp {
	resp := svc.Resp{
		"error": format.JSONPtr(err),
	}
	return resp
}

// OK is used to successfully respond with status 200.
func OK(
	ctx context.Context,
	w http.ResponseWriter,
	resp svc.Resp,
) {
	Respond(ctx, w, http.StatusOK, nil, resp)
}

// Created is used to successfully respond with status 201.
func Created(
	ctx context.Context,
	w http.ResponseWriter,
	resp svc.Resp,
) {
	Respond(ctx, w, http.StatusCreated, nil, resp)
}

// Error triage the error and respond with its content if it's a UserError,
// otherwise responds with a default `api_error`
func Error(
	ctx context.Context,
	w http.ResponseWriter,
	err error,
) {
	// Handle UserError
	if e := errors.ExtractUserError(err); e != nil {
		logging.Logf(ctx,
			"UserError: status=%d code=%q message=%q",
			e.Status(), e.Code(), e.Message())
		for _, line := range errors.ErrorStack(err) {
			logging.Logf(ctx, "  %s", line)
		}
		for _, line := range errors.ErrorStack(e.Cause()) {
			logging.Logf(ctx, "    %s", line)
		}

		resp := errorResponse(ctx, errors.Build(e))
		Respond(ctx, w, e.Status(), nil, resp)
	} else {
		logging.Logf(ctx,
			"Unexpected error: error=%q", err.Error())
		for _, line := range errors.ErrorStack(err) {
			logging.Logf(ctx, "  %s", line)
		}

		resp := errorResponse(ctx, errors.Build(panicError()))
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
	formatted, err := json.MarshalIndent(response, "", "  ")
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
