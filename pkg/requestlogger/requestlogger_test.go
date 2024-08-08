package requestlogger_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	md "github.com/go-chi/chi/middleware"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/nada-backend/pkg/requestlogger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type LogFormat struct {
	Level     string    `json:"level"`
	RequestID string    `json:"request_id"`
	Time      time.Time `json:"time"`
	BytesIn   int       `json:"bytes_in"`
	BytesOut  int       `json:"bytes_out"`
	Latency   float64   `json:"latency_ms"`
	Request   string    `json:"request"`
	Message   string    `json:"message"`
	Browser   string    `json:"browser"`
}

func TestLoggerMiddleware(t *testing.T) {
	testCases := []struct {
		name      string
		method    string
		target    string
		body      []byte
		userAgent string
		filters   []string
		expect    *LogFormat
	}{
		{
			name:      "Should work",
			method:    http.MethodGet,
			target:    "http://example.com/foo",
			body:      nil,
			userAgent: "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36",
			expect: &LogFormat{
				Level:    "info",
				BytesIn:  0,
				BytesOut: 2,
				Request:  "GET /foo (response_code: 200)",
				Message:  "incoming_request",
				Browser:  "Chrome (Windows)",
			},
		},
		{
			name:      "Should work with filters",
			method:    http.MethodGet,
			target:    "http://example.com/to-be-filtered",
			body:      nil,
			userAgent: "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36",
			filters:   []string{"/to-be-filtered"},
			expect:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			logger := zerolog.New(&buf)
			middleware := requestlogger.Middleware(logger, tc.filters...)

			req := httptest.NewRequest(tc.method, tc.target, bytes.NewReader(tc.body))
			req.Header.Set("User-Agent", tc.userAgent)
			w := httptest.NewRecorder()

			handler := md.RequestID(middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})))

			handler.ServeHTTP(w, req)

			if tc.expect == nil {
				assert.Empty(t, buf.String())
				return
			}

			got := &LogFormat{}
			err := json.Unmarshal(buf.Bytes(), got)
			require.NoError(t, err)

			fmt.Println(buf.String())

			diff := cmp.Diff(tc.expect, got, cmpopts.IgnoreFields(LogFormat{}, "Time", "Latency", "RequestID"))
			assert.Empty(t, diff)
			assert.Greater(t, got.Latency, 0.0)
			assert.GreaterOrEqual(t, got.Time.Unix(), time.Now().Unix())
			assert.NotEqual(t, "n/a", got.RequestID)
		})
	}
}
