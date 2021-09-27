package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/endpoints"

	"github.com/navikt/datakatalogen/backend/config"
)

const (
	AzureGraphMemberOfEndpoint = "https://graph.microsoft.com/v1.0/me/memberOf"
	CacheDuration              = 1 * time.Hour
)

type CacheEntry struct {
	groups  []string
	updated time.Time
}

type AzureGroups struct {
	Cache  map[string]CacheEntry
	Client *http.Client
	Config config.Config
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

type MemberOfResponse struct {
	Groups []MemberOfGroup `json:"value"`
}

type MemberOfGroup struct {
	Id string `json:"id"`
}

func (a *AzureGroups) GetGroupsForUser(ctx context.Context, token, email string) ([]string, error) {
	log.Tracef("Current cache: %v", a.Cache)

	entry, found := a.Cache[email]
	if found && entry.updated.Add(CacheDuration).After(time.Now()) {
		log.Debugf("Returning cached groups for user: %v", email)
		return entry.groups, nil
	}

	bearerToken, err := a.getBearerTokenOnBehalfOfUser(ctx, token)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AzureGraphMemberOfEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", bearerToken))
	response, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}

	var memberOfResponse MemberOfResponse
	if err := json.NewDecoder(response.Body).Decode(&memberOfResponse); err != nil {
		return nil, err
	}

	var groups []string
	for _, entry := range memberOfResponse.Groups {
		groups = append(groups, entry.Id)
	}

	a.Cache[email] = CacheEntry{
		groups:  groups,
		updated: time.Now(),
	}

	log.Tracef("Retrieved and cached groups: %v", groups)

	return groups, nil
}

func (a *AzureGroups) getBearerTokenOnBehalfOfUser(ctx context.Context, token string) (string, error) {

	form := url.Values{}
	form.Add("client_id", a.Config.OAuth2.ClientID)
	form.Add("client_secret", a.Config.OAuth2.ClientSecret)
	form.Add("scope", "https://graph.microsoft.com/.default")
	form.Add("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Add("requested_token_use", "on_behalf_of")
	form.Add("assertion", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoints.AzureAD(a.Config.OAuth2.TenantID).TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	response, err := a.Client.Do(req)
	if err != nil {
		return "", err
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&tokenResponse); err != nil {
		return "", err
	}

	log.Debugf("Successfully retrieved on-behalf-of token: %v...", tokenResponse.AccessToken[:5])
	return tokenResponse.AccessToken, nil
}
