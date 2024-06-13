package handlers

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type Tester interface {
	Invocations() int
	Reset()
}

type TestData struct {
	ID string `json:"id,omitempty"`
}

type testSimpleHandler struct {
	invocations int
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

func (h *testSimpleHandler) SimpleNoInput(_ context.Context, _ *http.Request, _ interface{}) (*TestData, error) {
	h.invocations++

	return &TestData{
		ID: "test",
	}, nil
}

func TestHandlerFor(t *testing.T) {

	simple := &testSimpleHandler{}

	testCases := []struct {
		name    string
		desc    string
		handler http.HandlerFunc
		path    string
		store   Tester
		request *http.Request
		status  int
	}{
		{
			name:    "handler-for-simple",
			desc:    "Just invokes the handler, so no output is expected",
			store:   simple,
			path:    "/test",
			handler: HandlerFor(simple.Simple).Build(),
			request: httptest.NewRequest(http.MethodGet, "/test", nil),
			status:  http.StatusOK,
		},
		{
			name:    "handler-for-json-response",
			desc:    "Invokes the handler and returns the response as JSON, expecting the result to be empty {}",
			store:   simple,
			path:    "/test",
			handler: HandlerFor(simple.Simple).ResponseToJSON().Build(),
			request: httptest.NewRequest(http.MethodGet, "/test", nil),
			status:  http.StatusOK,
		},
		{
			name:    "handler-for-json-request",
			desc:    "Invokes the handler and parses the request from JSON, expect it to work, no result in body",
			store:   simple,
			path:    "/test",
			handler: HandlerFor(simple.Simple).RequestFromJSON().Build(),
			request: httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(`{"id": "test"}`)),
			status:  http.StatusOK,
		},
		{
			name:    "handler-for-json-request-response",
			desc:    "Invokes the handler, parses the request from JSON and returns the response as JSON, expect it to work",
			store:   simple,
			path:    "/test",
			handler: HandlerFor(simple.Simple).RequestFromJSON().ResponseToJSON().Build(),
			request: httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(`{"id": "test"}`)),
			status:  http.StatusOK,
		},
		{
			name:    "handler-for-json-request-response-no-input",
			desc:    "Invokes the handler, parses the request from JSON and returns the response as JSON, expect it to work",
			store:   simple,
			path:    "/test",
			handler: HandlerFor(simple.SimpleNoInput).ResponseToJSON().Build(),
			request: httptest.NewRequest(http.MethodGet, "/test", nil),
			status:  http.StatusOK,
		},
		// FIXME: test all the possible error scenarios
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Get(tc.path, tc.handler)

			r.ServeHTTP(rr, tc.request)

			assert.Equal(t, tc.status, rr.Code)
			assert.Equal(t, 1, tc.store.Invocations())
			defer tc.store.Reset()

			g := goldie.New(t)
			g.Assert(t, tc.name, rr.Body.Bytes())
		})
	}
}
