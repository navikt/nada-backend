package requestlogger_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/navikt/nada-backend/pkg/requestlogger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type LogFormat struct {
	Level     string    `json:"level"`
	Time      time.Time `json:"time"`
	BytesIn   int       `json:"bytes_in"`
	BytesOut  int       `json:"bytes_out"`
	Latency   float64   `json:"latency_ms"`
	Method    string    `json:"method"`
	Proto     string    `json:"proto"`
	RemoteIP  string    `json:"remote_ip"`
	Status    int       `json:"status"`
	URL       string    `json:"url"`
	UserAgent string    `json:"user_agent"`
	Message   string    `json:"message"`
}

func TestLoggerMiddleware_LogsIncomingRequest(t *testing.T) {
	testCases := []struct {
		name       string
		method     string
		target     string
		body       []byte
		userAgent  string
		remoteAddr string
		filters    []string
		expect     *LogFormat
	}{
		{
			name:       "Should work",
			method:     http.MethodGet,
			target:     "http://example.com/foo",
			body:       nil,
			userAgent:  "test-agent",
			remoteAddr: "127.0.0.1:1234",
			expect: &LogFormat{
				Level:     "info",
				BytesIn:   0,
				BytesOut:  2,
				Method:    http.MethodGet,
				Proto:     "HTTP/1.1",
				RemoteIP:  "127.0.0.1:1234",
				Status:    http.StatusOK,
				URL:       "/foo",
				UserAgent: "test-agent",
				Message:   "incoming_request",
			},
		},
		{
			name:       "Should work with filters",
			method:     http.MethodGet,
			target:     "http://example.com/to-be-filtered",
			body:       nil,
			userAgent:  "test-agent",
			remoteAddr: "127.0.0.1:1234",
			filters:    []string{"/to-be-filtered"},
			expect:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			logger := zerolog.New(&buf)
			middleware := requestlogger.Middleware(logger, tc.filters...)

			req := httptest.NewRequest(tc.method, tc.target, bytes.NewReader(tc.body))
			req.Header.Set("User-Agent", tc.userAgent)
			req.RemoteAddr = tc.remoteAddr
			w := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))

			handler.ServeHTTP(w, req)

			if tc.expect == nil {
				assert.Empty(t, buf.String())
				return
			}

			got := &LogFormat{}
			fmt.Println(buf.String())
			err := json.Unmarshal(buf.Bytes(), got)
			assert.NoError(t, err)
			diff := cmp.Diff(tc.expect, got, cmpopts.IgnoreFields(LogFormat{}, "Time", "Latency"))
			assert.Empty(t, diff)
			assert.Greater(t, got.Latency, 0.0)
			assert.GreaterOrEqual(t, got.Time.Unix(), time.Now().Unix())
		})
	}
}
