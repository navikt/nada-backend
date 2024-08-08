package requestlogger

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mileusna/useragent"

	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
)

func Middleware(log zerolog.Logger, pathFilters ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			for _, filter := range pathFilters {
				if filter == r.URL.Path {
					next.ServeHTTP(w, r)
					return
				}
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				ua := useragent.Parse(r.UserAgent())

				requestID := middleware.GetReqID(r.Context())
				if requestID == "" {
					requestID = "n/a"
				}

				bytesIn, err := strconv.Atoi(r.Header.Get("Content-Length"))
				if err != nil {
					bytesIn = 0
				}

				log.Info().Timestamp().Fields(map[string]interface{}{
					"request_id": requestID,
					"request":    fmt.Sprintf("%v %v (response_code: %v)", r.Method, r.URL.Path, ww.Status()),
					"browser":    fmt.Sprintf("%v (%v)", ua.Name, ua.OS),
					"bytes_in":   bytesIn,
					"bytes_out":  ww.BytesWritten(),
					"latency_ms": float64(t2.Sub(t1).Nanoseconds()) / 1000000.0, //nolint: gomnd
				}).Msg("incoming_request")
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
