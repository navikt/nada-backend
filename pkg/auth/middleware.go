package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
)

type contextKey int

const contextUserKey contextKey = 1

type User struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Teams       []string
	AccessToken string `json:"-"`
}

func GetUser(ctx context.Context) *User {
	return ctx.Value(contextUserKey).(*User)
}

type teamsCache interface {
	Get(uuid string) (string, bool)
}

func JWTValidatorMiddleware(tokenVerifier *oidc.IDTokenVerifier) openapi.MiddlewareFunc { // discoveryURL, clientID string, azureGroups *AzureGroups, teamUUIDs teamsCache) openapi.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(openapi.CookieAuthScopes) == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Parse and verify ID Token payload.
			cookie, err := r.Cookie("jwt")
			if err != nil {
				http.Error(w, "uh oh", http.StatusForbidden)
				return
			}

			parts := strings.SplitN(cookie.Value, "|", 2)
			if len(parts) != 2 {
				http.Error(w, "uh oh", http.StatusForbidden)
				return
			}
			accessToken := parts[0]

			idToken, err := tokenVerifier.Verify(r.Context(), parts[1])
			if err != nil {
				http.Error(w, "uh oh", http.StatusForbidden)
				return
			}

			user := &User{
				AccessToken: accessToken,
			}
			if err := idToken.Claims(user); err != nil {
				logrus.WithError(err).Error("Unable to decode claims")
				http.Error(w, "uh oh", http.StatusForbidden)
				return
			}

			ctx := r.Context()
			r = r.WithContext(context.WithValue(ctx, contextUserKey, user))
			next.ServeHTTP(w, r)
		}
	}
}
