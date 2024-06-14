package handlers

import (
	"context"
	"encoding/json"
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
type EncoderFunc[Out any] func(w http.ResponseWriter, out Out) error

type HandlerConfig[In any, Out any] struct {
	decoderFn DecoderFunc[In]
	encoderFn EncoderFunc[Out]
	targetFn  TargetFunc[In, Out]
}

func HandlerFor[In any, Out any](target TargetFunc[In, Out]) *HandlerConfig[In, Out] {
	return &HandlerConfig[In, Out]{
		targetFn: target,
	}
}

func (h *HandlerConfig[In, Out]) RequestFromJSON() *HandlerConfig[In, Out] {
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

func (h *HandlerConfig[In, Out]) ResponseToJSON() *HandlerConfig[In, Out] {
	h.encoderFn = func(w http.ResponseWriter, out Out) error {
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

	return h
}

func (h *HandlerConfig[In, Out]) Build() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in In
		var err error

		if h.decoderFn != nil {
			in, err = h.decoderFn(r)
			if err != nil {
				// Handle error, probably just logging
				return
			}
		}

		if in != nil {
			if v, ok := any(in).(Validator); ok {
				err := v.Validate()
				if err != nil {
					// Format error response, something went wrong inside our app
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
		}

		out, err := h.targetFn(r.Context(), r, in)
		// FIXME: handle error
		if err != nil {
			// Format error response, something went wrong inside our app
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Format and write response
		if h.encoderFn != nil {
			err = h.encoderFn(w, out)
			if err != nil {
				// Handle error, probably just logging
				return
			}
		}
	}
}
