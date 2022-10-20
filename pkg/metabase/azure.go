package metabase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) GetAzureGroupID(ctx context.Context, email string) (string, error) {
	token, err := c.getAzureAccessToken(ctx)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/v1.0/groups", nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	q.Add("$filter", fmt.Sprintf("startswith(mail, '%v')", email))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("ConsistencyLevel", "eventual")
	res, err := c.c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	type groupRes struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	group := &groupRes{}

	if err := json.NewDecoder(res.Body).Decode(group); err != nil {
		return "", err
	}

	if len(group.Value) != 1 {
		return "", fmt.Errorf("unable to find azure group with email %v", email)
	}

	return group.Value[0].ID, nil
}

func (c *Client) getAzureAccessToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	form.Add("client_id", c.oauth2ClientID)
	form.Add("client_secret", c.oauth2ClientSecret)
	form.Add("scope", "https://graph.microsoft.com/.default")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/oauth2/v2.0/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Keep-Alive", "true")
	res, err := c.c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	tokenRes := &tokenResponse{}
	if err := json.NewDecoder(res.Body).Decode(tokenRes); err != nil {
		return "", err
	}

	return tokenRes.AccessToken, nil
}
