// Package tk provides a client for the team catalog API.
// - https://teamkatalog-api.intern.nav.no/swagger-ui/index.html
package tk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

const (
	ConsumerIDHeader = "Nav-Consumer-Id"
	ConsumerID       = "nada-backend"
)

type Fetcher interface {
	GetProductAreas(ctx context.Context) (*ProductAreas, error)
	GetTeams(ctx context.Context) (*Teams, error)
	GetTeam(ctx context.Context, teamID uuid.UUID) (*Team, error)
	GetTeamsInProductArea(ctx context.Context, productAreaID uuid.UUID) (*Teams, error)
	GetTeamCatalogURL(teamID uuid.UUID) string
}

type Client struct {
	client *http.Client
	apiURL string
}

type Teams struct {
	Content []Team `json:"content"`
}

type Links struct {
	UI string `json:"ui"`
}

type Team struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Links         Links     `json:"links"`
	NaisTeams     []string  `json:"naisTeams"`
	ProductAreaID uuid.UUID `json:"productAreaId"`
}

type ProductAreas struct {
	Content []ProductArea `json:"content"`
}

type ProductArea struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	AreaType string    `json:"areaType"`
}

func (c *Client) GetTeamCatalogURL(teamID uuid.UUID) string {
	return fmt.Sprintf("%s/team/%s", c.apiURL, teamID.String())
}

func (c *Client) GetTeams(ctx context.Context) (*Teams, error) {
	url := fmt.Sprintf("%s/team?status=ACTIVE", c.apiURL)

	teams := &Teams{}
	err := c.sendRequestAndDeserialize(ctx, http.MethodGet, url, &teams)
	if err != nil {
		return nil, err
	}

	return teams, nil
}

func (c *Client) GetTeamsInProductArea(ctx context.Context, productAreaID uuid.UUID) (*Teams, error) {
	url := fmt.Sprintf("%s/team?status=ACTIVE&productAreaId=%s", c.apiURL, productAreaID.String())

	teams := &Teams{}
	err := c.sendRequestAndDeserialize(ctx, http.MethodGet, url, &teams)
	if err != nil {
		return nil, err
	}

	return teams, nil
}

func (c *Client) GetTeam(ctx context.Context, teamID uuid.UUID) (*Team, error) {
	url := fmt.Sprintf("%s/team/%s", c.apiURL, teamID.String())

	team := &Team{}
	err := c.sendRequestAndDeserialize(ctx, http.MethodGet, url, team)
	if err != nil {
		return nil, err
	}

	return team, nil
}

func (c *Client) GetProductAreas(ctx context.Context) (*ProductAreas, error) {
	url := fmt.Sprintf("%s/productarea?status=ACTIVE", c.apiURL)

	productAreas := &ProductAreas{}
	err := c.sendRequestAndDeserialize(ctx, http.MethodGet, url, productAreas)
	if err != nil {
		return nil, err
	}

	return productAreas, nil
}

func (c *Client) sendRequestAndDeserialize(ctx context.Context, method, url string, into any) error {
	req, err := c.newRequestWithHeaders(ctx, method, url)
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

func (c *Client) newRequestWithHeaders(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set(ConsumerIDHeader, ConsumerID)

	return req, nil
}

func New(apiURL string, client *http.Client) *Client {
	return &Client{
		apiURL: apiURL,
		client: client,
	}
}
