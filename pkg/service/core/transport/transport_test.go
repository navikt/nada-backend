package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
)

type TestData struct {
	ID string `json:"id,omitempty"`
}

type testSimpleHandler struct {
	invocations int
	Data        []byte
	NewURL      string
	Request     *http.Request
}

func (h *testSimpleHandler) Reset() {
	h.invocations = 0
}

func (h *testSimpleHandler) Invocations() int {
	return h.invocations
}

func (h *testSimpleHandler) Simple(_ context.Context, _ *http.Request, in TestData) (*TestData, error) {
	h.invocations++

	return &TestData{
		ID: in.ID,
	}, nil
}

func (h *testSimpleHandler) SimpleNoInput(_ context.Context, _ *http.Request, _ any) (*TestData, error) {
	h.invocations++

	return &TestData{
		ID: "test",
	}, nil
}

func (h *testSimpleHandler) SimpleNoOutput(_ context.Context, _ *http.Request, in TestData) (*Empty, error) {
	h.invocations++

	return &Empty{}, nil
}

func (h *testSimpleHandler) ParamFromContext(ctx context.Context, _ *http.Request, _ any) (*TestData, error) {
	h.invocations++

	return &TestData{
		ID: chi.URLParamFromCtx(ctx, "id"),
	}, nil
}

func (h *testSimpleHandler) ByteWriterEncoder(_ context.Context, _ *http.Request, _ any) (*ByteWriter, error) {
	h.invocations++

	return NewByteWriter("text/plain", "utf-8", h.Data), nil
}

func (h *testSimpleHandler) RedirectEncoder(_ context.Context, r *http.Request, _ any) (*Redirect, error) {
	h.invocations++

	return NewRedirect(h.NewURL, r), nil
}

func (h *testSimpleHandler) Receiver(_ context.Context, _ *http.Request, _ any) (*TestData, error) {
	h.invocations++

	return &TestData{
		ID: "I was redirected",
	}, nil
}

func (h *testSimpleHandler) Accepted(_ context.Context, _ *http.Request, _ any) (*Accepted, error) {
	h.invocations++

	return &Accepted{}, nil
}

func TestHandlerFor(t *testing.T) {
	simple := &testSimpleHandler{
		Data:   []byte("test"),
		NewURL: "/receiver",
	}

	logger := zerolog.New(os.Stdout)

	testCases := []struct {
		name    string
		desc    string
		routes  map[string]http.HandlerFunc
		request *http.Request
		status  int
		count   int
	}{
		{
			name: "handler-for-json-response",
			desc: "Invokes the handler and returns the response as JSON, expecting the result to be empty {}",
			routes: map[string]http.HandlerFunc{
				"/test": For(simple.Simple).Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/test", nil),
			status:  http.StatusOK,
			count:   1,
		},
		{
			name: "handler-for-json-request-response",
			desc: "Invokes the handler, parses the request from JSON and returns the response as JSON, expect it to work",
			routes: map[string]http.HandlerFunc{
				"/test": For(simple.Simple).RequestFromJSON().Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(`{"id": "test"}`)),
			status:  http.StatusOK,
			count:   1,
		},
		{
			name: "handler-for-json-request-response-no-input",
			desc: "Invokes the handler, parses the request from JSON and returns the response as JSON, expect it to work",
			routes: map[string]http.HandlerFunc{
				"/test": For(simple.SimpleNoInput).Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/test", nil),
			status:  http.StatusOK,
			count:   1,
		},
		{
			name: "handler-for-json-request-response-no-output",
			desc: "Invokes the handler, parses the request from JSON and returns the response as JSON, expect it to work",
			routes: map[string]http.HandlerFunc{
				"/test": For(simple.SimpleNoOutput).Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(`{"id": "test"}`)),
			status:  http.StatusNoContent,
			count:   1,
		},
		{
			name: "handler-for-param-from-context",
			desc: "Invokes the handler and expects the parameter to be taken from the context",
			routes: map[string]http.HandlerFunc{
				"/test/{id}": For(simple.ParamFromContext).Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/test/123", nil),
			status:  http.StatusOK,
			count:   1,
		},
		{
			name: "handler-for-bytewriter-encoder",
			desc: "Invokes the handler and expects the custom encoder to be used",
			routes: map[string]http.HandlerFunc{
				"/whatever": For(simple.ByteWriterEncoder).Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/whatever", nil),
			status:  http.StatusOK,
			count:   1,
		},
		{
			name: "handler-for-redirect-encoder",
			desc: "Invokes the handler and expects the custom encoder to be used",
			routes: map[string]http.HandlerFunc{
				"/whatever": For(simple.RedirectEncoder).Build(logger),
				"/receiver": For(simple.Receiver).Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/whatever", nil),
			status:  http.StatusSeeOther,
			count:   2,
		},
		{
			name: "handler-for-accepted-encoder",
			desc: "Invokes the handler and expects the custom encoder to be used",
			routes: map[string]http.HandlerFunc{
				"/whatever": For(simple.Accepted).Build(logger),
			},
			request: httptest.NewRequest(http.MethodGet, "/whatever", nil),
			status:  http.StatusAccepted,
			count:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			r := chi.NewRouter()
			for path, handler := range tc.routes {
				r.Get(path, handler)
			}

			r.ServeHTTP(rr, tc.request)

			if rr.Code == http.StatusSeeOther {
				r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, rr.Header().Get("Location"), nil))
			}

			assert.Equal(t, tc.status, rr.Code)
			assert.Equal(t, tc.count, simple.Invocations())
			defer simple.Reset()

			g := goldie.New(t)
			g.Assert(t, tc.name, rr.Body.Bytes())
		})
	}
}
