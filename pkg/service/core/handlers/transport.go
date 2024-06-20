package handlers

import (
	"context"
	"encoding/json"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/rs/zerolog"
	"net/http"
)

// Inspired by: https://www.willem.dev/articles/generic-http-handlers/
// go-kit
// grafana matt article

type StatusCoder interface {
	StatusCode() int
}

type Validator interface {
	Validate() error
}

// Empty provides a convenience struct for returning an empty response
type Empty struct{}

func (e *Empty) StatusCode() int {
	return http.StatusNoContent
}

type DecoderFunc[In any] func(r *http.Request) (In, error)

// We can probably get rid of having to pass the Request to the target function
// just need to move some query params to chi context
type TargetFunc[In any, Out any] func(context.Context, *http.Request, In) (Out, error)

type TransportConfig[In any, Out any] struct {
	decoderFn DecoderFunc[In]
	targetFn  TargetFunc[In, Out]
}

func TransportFor[In any, Out any](target TargetFunc[In, Out]) *TransportConfig[In, Out] {
	return &TransportConfig[In, Out]{
		targetFn: target,
	}
}

func (h *TransportConfig[In, Out]) RequestFromJSON() *TransportConfig[In, Out] {
	h.decoderFn = func(r *http.Request) (In, error) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			return in, err
		}

		return in, nil
	}

	return h
}

func (h *TransportConfig[In, Out]) encode(w http.ResponseWriter, out Out) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// If the output implements the StatusCoder interface, use the status code from it
	code := http.StatusOK
	if sc, ok := any(out).(StatusCoder); ok {
		code = sc.StatusCode()
	}

	w.WriteHeader(code)
	if code == http.StatusNoContent {
		return nil
	}

	err := json.NewEncoder(w).Encode(out)
	if err != nil {
		return err
	}

	return nil
}

func (h *TransportConfig[In, Out]) Build(logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info().Str("method", r.Method).Str("url", r.URL.RequestURI())

		var in In
		var err error

		if h.decoderFn != nil {
			in, err = h.decoderFn(r)
			if err != nil {
				errs.HTTPErrorResponse(w, logger, errs.E(errs.InvalidRequest, err))
				return
			}
		}

		if v, ok := any(in).(Validator); ok {
			err := v.Validate()
			if err != nil {
				errs.HTTPErrorResponse(w, logger, errs.E(errs.Validation, err))
				return
			}
		}

		out, err := h.targetFn(r.Context(), r, in)
		if err != nil {
			errs.HTTPErrorResponse(w, logger, err)
			return
		}

		// We always encode the response as JSON, you can use Empty{} or build
		// a custom reponse struct if needed
		err = h.encode(w, out)
		if err != nil {
			errs.HTTPErrorResponse(w, logger, errs.E(errs.Internal, err))
			return
		}
	}
}
