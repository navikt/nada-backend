package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/service"
	"github.com/rs/zerolog"

	"golang.org/x/oauth2/endpoints"
)

const (
	AzureGraphMemberOfEndpoint = "https://graph.microsoft.com/v1.0/me/memberOf/microsoft.graph.group?$select=mail,groupTypes,displayName"
	CacheDuration              = 1 * time.Hour
)

type AzureGroupClient struct {
	Client            *http.Client
	OAuthClientID     string
	OAuthClientSecret string
	OAuthTenantID     string
	log               zerolog.Logger
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

type MemberOfResponse struct {
	Groups []MemberOfGroup `json:"value"`
}

type MemberOfGroup struct {
	DisplayName string   `json:"displayName"`
	Mail        string   `json:"mail"`
	GroupTypes  []string `json:"groupTypes"`
}

func NewAzureGroups(client *http.Client, clientID, clientSecret, tenantID string, log zerolog.Logger) *AzureGroupClient {
	return &AzureGroupClient{
		Client:            client,
		OAuthClientID:     clientID,
		OAuthClientSecret: clientSecret,
		OAuthTenantID:     tenantID,
		log:               log,
	}
}

func (a *AzureGroupClient) GroupsForUser(ctx context.Context, token, email string) (service.Groups, error) {
	bearerToken, err := a.getBearerTokenOnBehalfOfUser(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("getting bearer token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AzureGraphMemberOfEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", bearerToken))
	response, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}

	var memberOfResponse MemberOfResponse
	if err := json.NewDecoder(response.Body).Decode(&memberOfResponse); err != nil {
		return nil, err
	}
	var groups service.Groups

	for _, entry := range memberOfResponse.Groups {
		mail := strings.ToLower(entry.Mail)
		if !contains("Unified", entry.GroupTypes) || !strings.HasSuffix(mail, "@nav.no") {
			continue
		}
		groups = append(groups, service.Group{
			Name:  entry.DisplayName,
			Email: mail,
		})
	}

	return groups, nil
}

func (a *AzureGroupClient) getBearerTokenOnBehalfOfUser(ctx context.Context, token string) (string, error) {
	form := url.Values{}
	form.Add("client_id", a.OAuthClientID)
	form.Add("client_secret", a.OAuthClientSecret)
	form.Add("scope", "https://graph.microsoft.com/.default")
	form.Add("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Add("requested_token_use", "on_behalf_of")
	form.Add("assertion", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoints.AzureAD(a.OAuthTenantID).TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	response, err := a.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("performing request: %w", err)
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	a.log.Debug().Msg("Successfully retrieved on-behalf-of token")
	return tokenResponse.AccessToken, nil
}

func contains(elem string, list []string) bool {
	for _, entry := range list {
		if entry == elem {
			return true
		}
	}
	return false
}
