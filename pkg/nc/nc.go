// Package nc implements a simple client for interacting with the Nais console API
package nc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Fetcher interface {
	GetTeamGoogleProjects(ctx context.Context) (map[string]string, error)
}

type Client struct {
	client          *http.Client
	apiURL          string
	apiKey          string
	naisClusterName string
}

type Team struct {
	Slug         string        `json:"slug"`
	Environments []Environment `json:"environments"`
}

type Environment struct {
	Name         string `json:"name"`
	GcpProjectID string `json:"gcpProjectID"`
}

type Response struct {
	Data Data `json:"data"`
}

type Data struct {
	Teams Teams `json:"teams"`
}

type Teams struct {
	Nodes    []Team   `json:"nodes"`
	PageInfo PageInfo `json:"pageInfo"`
}

type PageInfo struct {
	HasNextPage bool `json:"hasNextPage"`
}

func (c *Client) GetTeamGoogleProjects(ctx context.Context) (map[string]string, error) {
	gqlQuery := `
		query GCPTeams($limit: Int, $offset: Int){
			teams(limit: $limit, offset: $offset) {
				nodes {
					slug
					environments {
						name
						gcpProjectID
					}
				}
				pageInfo {
					hasNextPage
				}
			}
		}
	`

	const limit = 100
	offset := 0

	mapping := map[string]string{}

	for {
		payload := map[string]any{
			"query": gqlQuery,
			"variables": map[string]any{
				"limit":  limit,
				"offset": offset,
			},
		}

		r := Response{}

		err := c.sendRequestAndDeserialize(ctx, http.MethodPost, "/query", payload, &r)
		if err != nil {
			return nil, fmt.Errorf("fetching team google projects: %w", err)
		}

		for _, team := range r.Data.Teams.Nodes {
			for _, env := range team.Environments {
				if env.Name == c.naisClusterName {
					mapping[team.Slug] = env.GcpProjectID
				}
			}
		}

		if !r.Data.Teams.PageInfo.HasNextPage {
			break
		}

		offset += limit
	}

	return mapping, nil
}

func (c *Client) sendRequestAndDeserialize(ctx context.Context, method, url string, body, into any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling body: %w", err)
	}

	req, err := c.newRequestWithHeaders(ctx, method, url, data)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	err = json.NewDecoder(res.Body).Decode(into)
	if err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

func (c *Client) newRequestWithHeaders(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	return req, nil
}

func New(apiURL, apiKey, naisClusterName string, client *http.Client) *Client {
	return &Client{
		client:          client,
		apiURL:          apiURL,
		apiKey:          apiKey,
		naisClusterName: naisClusterName,
	}
}
