package auth

import (
	"context"
	"net/http"

	"github.com/navikt/nada-backend/pkg/metadata"
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

type MiddlewareHandler func(http.Handler) http.Handler

func MockJWTValidatorMiddleware() MiddlewareHandler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-NO-AUTH") != "" {
				next.ServeHTTP(w, r)
				return
			}
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
