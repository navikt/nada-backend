package auth

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/openapi"
)

var MockUser = User{
	Name:  "Anderson, Mock",
	Email: "mock.anderson@email.com",
	Teams: []string{"team"},
}

func MockJWTValidatorMiddleware() openapi.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			teams := []string{
				"team",
			}
			if mockTeam := r.Header.Get("X-Mock-Team"); mockTeam != "" {
				teams[0] = mockTeam
			}

			ctx := context.WithValue(r.Context(), contextUserKey, &MockUser)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
