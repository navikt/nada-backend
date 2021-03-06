package teamkatalogen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/navikt/nada-backend/pkg/graph/models"
)

type Teamkatalogen struct {
	client *http.Client
	url    string
}

type TeamkatalogenResponse struct {
	Content []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Links       struct {
			Ui string `json:"ui"`
		} `json:"links"`
		NaisTeams []string `json:"naisTeams"`
	} `json:"content"`
}

func New(url string) *Teamkatalogen {
	return &Teamkatalogen{client: http.DefaultClient, url: url}
}

func (t *Teamkatalogen) Search(ctx context.Context, query string) ([]*models.TeamkatalogenResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/team", t.url), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	res, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}

	var tkRes TeamkatalogenResponse
	if err := json.NewDecoder(res.Body).Decode(&tkRes); err != nil {
		return nil, err
	}

	ret := []*models.TeamkatalogenResult{}
	for _, r := range tkRes.Content {
		isMatch := false
		if strings.Contains(strings.ToLower(r.Name), strings.ToLower(query)) {
			isMatch = true
		}
		for _, team := range r.NaisTeams {
			if strings.Contains(strings.ToLower(team), strings.ToLower(query)) {
				isMatch = true
				break
			}
		}
		if isMatch {
			ret = append(ret, &models.TeamkatalogenResult{
				URL:         r.Links.Ui,
				Name:        r.Name,
				Description: r.Description,
			})
		}
	}

	return ret, nil
}
