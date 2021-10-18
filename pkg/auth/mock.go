package auth

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/metadata"
	"github.com/navikt/nada-backend/pkg/openapi"
)

var MockUser = User{
	Name:  "Anderson, Mock",
	Email: "mock.anderson@email.com",
	Groups: metadata.Groups{
		{
			Name:  "team",
			Email: "team@nav.no",
		},
		{
			Name:  "dataplattform",
			Email: "dataplattform@nav.no",
		},
		{
			Name:  "aura",
			Email: "aura@nav.no",
		},
	},
}

var MockProjectIDs = []string{"team-dev", "team-prod"}

var MockTeamProjectsUpdater = TeamProjectsUpdater{
	teamProjects: map[string][]string{
		"team@nav.no":          MockProjectIDs,
		"dataplattform@nav.no": {"dataplattform-dev-9da3"},
		"aura@nav.no":          {"aura-dev-d9f5"},
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
