package auth

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/openapi"
)

var MockUser = User{
	Name:  "Anderson, Mock",
	Email: "mock.anderson@email.com",
	Teams: []string{"team", "dataplattform", "aura"},
}

var MockProjectIDs = []string{"team-dev", "team-prod"}

var MockTeamProjectsUpdater = TeamProjectsUpdater{
	teamProjects: map[string][]string{
		"team":          MockProjectIDs,
		"dataplattform": {"dataplattform-dev-9da3"},
		"aura":          {"aura-dev-d9f5"},
	},
}

func MockJWTValidatorMiddleware() openapi.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			teams := []string{
				"team",
			}
			if mockTeam := r.Header.Get("X-Mock-Team"); mockTeam != "" {
				teams[0] = mockTeam
			}

			ctx := context.WithValue(r.Context(), contextUserKey, &MockUser)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}
	}
}
