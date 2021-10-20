package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/navikt/nada-backend/pkg/openapi"
)

type Google struct {
	oauth2.Config

	clientID     string
	clientSecret string
	hostname     string

	provider *oidc.Provider
	// teamUUIDs map[string]string
}

func NewGoogle(clientID, clientSecret, hostname string) *Google {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		panic(err)
	}

	g := &Google{
		clientID:     clientID,
		clientSecret: clientSecret,
		hostname:     hostname,
		provider:     provider,
	}
	g.setupOAuth2()
	return g
}

func (a *Google) setupOAuth2() {
	var callbackURL string
	if a.hostname == "localhost" {
		callbackURL = "http://localhost:8080/api/oauth2/callback"
	} else {
		callbackURL = fmt.Sprintf("https://%v/api/oauth2/callback", a.hostname)
	}

	a.Config = oauth2.Config{
		ClientID:     a.clientID,
		ClientSecret: a.clientSecret,
		Endpoint:     a.provider.Endpoint(),
		RedirectURL:  callbackURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "https://www.googleapis.com/auth/cloudplatformprojects.readonly"},
	}
}

func (g *Google) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return g.provider.Verifier(&oidc.Config{ClientID: g.clientID}).Verify(ctx, rawIDToken)
}

func (g *Google) Middleware(groupsLister GroupsLister) openapi.MiddlewareFunc {
	return newMiddleware(g.provider.Verifier(&oidc.Config{ClientID: g.clientID}), groupsLister).handle
}

// func (a *Google) Groups(client *http.Client) *GoogleGroups {
// 	return NewGoogleGroups(client, a.clientID, a.clientSecret, a.tenantID)
// }

// func (a *Google) Middleware(teamsCache teamsCache) openapi.MiddlewareFunc {
// 	return JWTValidatorMiddleware(a.KeyDiscoveryURL(), a.clientID, a.Groups(http.DefaultClient), teamsCache)
// }
