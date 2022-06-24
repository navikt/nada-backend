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

func NewAzureGroups(client *http.Client, clientID, clientSecret, tenantID string) *AzureGroupClient {
	return &AzureGroupClient{
		Client:            client,
		OAuthClientID:     clientID,
		OAuthClientSecret: clientSecret,
		OAuthTenantID:     tenantID,
	}
}

func (a *AzureGroupClient) GroupsForUser(ctx context.Context, token, email string) (Groups, error) {
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

	var groups Groups

	for _, entry := range memberOfResponse.Groups {
		mail := strings.ToLower(entry.Mail)
		if !contains("Unified", entry.GroupTypes) || !strings.HasSuffix(mail, "@nav.no") {
			continue
		}
		groups = append(groups, Group{
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
