package auth

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"golang.org/x/oauth2"
)

type Azure struct {
	oauth2.Config

	clientID     string
	clientSecret string
	clientTenant string
	hostname     string

	provider *oidc.Provider
}

func NewAzure(clientID, clientSecret, clientTenant, hostname string) *Azure {
	provider, err := oidc.NewProvider(context.Background(), fmt.Sprintf("https://login.microsoftonline.com/%v/v2.0", clientTenant))
	if err != nil {
		panic(err)
	}

	a := &Azure{
		clientID:     clientID,
		clientSecret: clientSecret,
		clientTenant: clientTenant,
		hostname:     hostname,
		provider:     provider,
	}
	a.setupOAuth2()
	return a
}

func (a *Azure) setupOAuth2() {
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
		Scopes:       []string{"openid", fmt.Sprintf("%s/.default", a.clientID)},
	}
}

func (a *Azure) KeyDiscoveryURL() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", a.clientTenant)
}

func (a *Azure) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return a.provider.Verifier(&oidc.Config{ClientID: a.clientID}).Verify(ctx, rawIDToken)
}

func (a *Azure) Middleware(keyDiscoveryURL string, azureGroups *AzureGroupClient, googleGroups *GoogleGroupClient, db *sql.DB) MiddlewareHandler {
	return newMiddleware(keyDiscoveryURL, a.provider.Verifier(&oidc.Config{ClientID: a.clientID}), azureGroups, googleGroups, gensql.New(db)).handle
}

// func (a *Google) Groups(client *http.Client) *GoogleGroups {
// 	return NewGoogleGroups(client, a.clientID, a.clientSecret, a.tenantID)
// }

// func (a *Google) Middleware(teamsCache teamsCache) openapi.MiddlewareFunc {
// 	return JWTValidatorMiddleware(a.KeyDiscoveryURL(), a.clientID, a.Groups(http.DefaultClient), teamsCache)
// }
