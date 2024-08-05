package requestlogger

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
)

func Middleware(logger zerolog.Logger, pathFilters ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			for _, filter := range pathFilters {
				if filter == r.URL.Path {
					next.ServeHTTP(w, r)
					return
				}
			}

			log := logger.With().Logger()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				bytesIn, err := strconv.Atoi(r.Header.Get("Content-Length"))
				if err != nil {
					bytesIn = 0
				}

				log.Info().Timestamp().Fields(map[string]interface{}{
					"remote_ip":  r.RemoteAddr,
					"url":        r.URL.Path,
					"proto":      r.Proto,
					"method":     r.Method,
					"user_agent": r.Header.Get("User-Agent"),
					"status":     ww.Status(),
					"latency_ms": float64(t2.Sub(t1).Nanoseconds()) / 1000000.0,
					"bytes_in":   bytesIn,
					"bytes_out":  ww.BytesWritten(),
				}).Msg("incoming_request")
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
