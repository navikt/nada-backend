package auth

import (
	"context"
	"net/http"
	"time"
)

var MockUser = User{
	Name:  "Anderson, Mock",
	Email: "mock.anderson@email.com",
	GoogleGroups: Groups{
		{
			Name:  "team frifrokost",
			Email: "teamfrifrokost@nav.no",
		},
		{
			Name:  "team frilunsj",
			Email: "teamfrilunsj@nav.no",
		},
		{
			Name:  "team frimiddag",
			Email: "teamfrimiddag@nav.no",
		},
		{
			Name:  "team frivin",
			Email: "teamfrivin@nav.no",
		},
		{
			Name:  "team friøl for person med veldig lang navn som denne",
			Email: "teamfriøl@nav.no",
		},
	},
	AzureGroups: Groups{
		{
			Name:  "team",
			Email: "team@nav.no",
		},
	},
	Expiry: time.Now().Add(time.Hour * 24),
}

type MiddlewareHandler func(http.Handler) http.Handler

func MockJWTValidatorMiddleware() MiddlewareHandler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-NO-AUTH") != "" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserKey, &MockUser)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
