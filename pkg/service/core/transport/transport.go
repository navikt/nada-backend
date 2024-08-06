// Package transport provides a generic HTTP transport layer for services.
//
// Inspired by:
// - https://www.willem.dev/articles/generic-http-handlers/ - for use of generics
// - https://github.com/go-kit/kit - for StatusCoder interface

package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/rs/zerolog"
)

type StatusCoder interface {
	StatusCode() int
}

type Encoder interface {
	Encode(w http.ResponseWriter) error
}

// DecoderFunc is a function that decodes a request into a struct
type DecoderFunc[In any] func(r *http.Request) (In, error)

// TargetFunc is a function that handles the request and returns a response, ideally
// we shouldn't have to use the http.Request, but sometimes we need it to fetch
// query parameters, headers, or similar
type TargetFunc[In any, Out any] func(context.Context, *http.Request, In) (Out, error)

type Transport[In any, Out any] struct {
	decoderFn DecoderFunc[In]
	targetFn  TargetFunc[In, Out]
}

func For[In any, Out any](target TargetFunc[In, Out]) *Transport[In, Out] {
	return &Transport[In, Out]{
		targetFn: target,
	}
}

func (h *Transport[In, Out]) RequestFromJSON() *Transport[In, Out] {
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

func (h *Transport[In, Out]) encode(w http.ResponseWriter, out Out) error {
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

func (h *Transport[In, Out]) Build(logger zerolog.Logger) http.HandlerFunc {
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

		out, err := h.targetFn(r.Context(), r, in)
		if err != nil {
			errs.HTTPErrorResponse(w, logger, err)
			return
		}

		// If the output implements the Encoder interface, use it
		if v, ok := any(out).(Encoder); ok {
			err := v.Encode(w)
			if err != nil {
				errs.HTTPErrorResponse(w, logger, errs.E(errs.Internal, err))
				return
			}

			return
		}

		// By default, we always encode the response as JSON, you can use
		// the Encoder or StatusCoder interfaces to customize the response
		err = h.encode(w, out)
		if err != nil {
			errs.HTTPErrorResponse(w, logger, errs.E(errs.Internal, err))
			return
		}
	}
}

type Redirect struct {
	newURL string
	r      *http.Request
}

func (r *Redirect) Encode(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "")
	http.Redirect(w, r.r, r.newURL, http.StatusSeeOther)
	return nil
}

func NewRedirect(newURL string, r *http.Request) *Redirect {
	return &Redirect{
		newURL: newURL,
		r:      r,
	}
}

// Empty provides a convenience struct for returning an empty response
type Empty struct{}

func (e *Empty) StatusCode() int {
	return http.StatusNoContent
}

// ByteWriter provides a convenience struct for returning a byte slice as a response
type ByteWriter struct {
	data            []byte
	contentType     string
	contentEncoding string
}

func (b *ByteWriter) Encode(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", b.contentType)
	w.Header().Set("Content-Encoding", b.contentEncoding)
	w.Header().Set("Content-Length", strconv.Itoa(len(b.data)))

	_, err := w.Write(b.data)
	if err != nil {
		return err
	}

	return nil
}

func NewByteWriter(typ, encoding string, data []byte) *ByteWriter {
	return &ByteWriter{
		data:            data,
		contentType:     typ,
		contentEncoding: encoding,
	}
}

type Accepted struct{}

func (a *Accepted) StatusCode() int {
	return http.StatusAccepted
}
