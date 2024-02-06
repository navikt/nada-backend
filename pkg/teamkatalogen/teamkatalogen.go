package teamkatalogen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/navikt/nada-backend/pkg/httpwithcache"
)

type Teamkatalogen struct {
	client *http.Client
	url    string
}

type TeamkatalogenResponse struct {
	Content []struct {
		TeamID      string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Links       struct {
			Ui string `json:"ui"`
		} `json:"links"`
		NaisTeams     []string `json:"naisTeams"`
		ProductAreaID string   `json:"productAreaId"`
	} `json:"content"`
}

func New(url string) *Teamkatalogen {
	return &Teamkatalogen{client: http.DefaultClient, url: url}
}

func (t *Teamkatalogen) Search(ctx context.Context, query string) ([]*models.TeamkatalogenResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/team?status=ACTIVE", t.url), nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, err
	}

	var tkRes TeamkatalogenResponse
	if err := json.NewDecoder(bytes.NewReader(res)).Decode(&tkRes); err != nil {
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
				URL:           r.Links.Ui,
				Name:          r.Name,
				Description:   r.Description,
				ProductAreaID: r.ProductAreaID,
				TeamID:        r.TeamID,
			})
		}
	}

	return ret, nil
}

func (t *Teamkatalogen) GetTeamsInProductArea(ctx context.Context, paID string) ([]*models.Team, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/team?status=ACTIVE&productAreaId="+paID, nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, err
	}

	var teams struct {
		Content []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ProductAreaID string `json:"productAreaId"`
		} `json:"content"`
	}

	if err := json.Unmarshal(res, &teams); err != nil {
		return nil, err
	}

	teamsGraph := make([]*models.Team, len(teams.Content))
	for idx, t := range teams.Content {
		teamsGraph[idx] = &models.Team{
			ID:            t.ID,
			Name:          t.Name,
			ProductAreaID: t.ProductAreaID,
		}
	}

	return teamsGraph, nil
}

func (t *Teamkatalogen) GetProductArea(ctx context.Context, paID string) (*models.ProductArea, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/productarea/"+paID, nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, err
	}

	var pa struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(res, &pa); err != nil {
		return nil, err
	}
	return &models.ProductArea{
		ID:   pa.ID,
		Name: pa.Name,
	}, nil
}

func (t *Teamkatalogen) GetProductAreas(ctx context.Context) ([]*models.ProductArea, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/productarea", nil)
	q := req.URL.Query()
	q.Add("status", "active")
	req.URL.RawQuery = q.Encode()

	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, err
	}

	var pasdto struct {
		Content []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			AreaType string `json:"areaType"`
		} `json:"content"`
	}

	if err := json.Unmarshal(res, &pasdto); err != nil {
		return nil, err
	}

	pas := make([]*models.ProductArea, 0)
	for _, pa := range pasdto.Content {
		pas = append(pas, &models.ProductArea{
			ID:       pa.ID,
			Name:     pa.Name,
			AreaType: pa.AreaType,
		})
	}

	return pas, nil
}

func (t *Teamkatalogen) GetTeam(ctx context.Context, teamID string) (*models.Team, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/team/"+teamID, nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, err
	}

	var team struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ProductAreaID string `json:"productAreaId"`
	}

	if err := json.Unmarshal(res, &team); err != nil {
		return nil, err
	}

	return &models.Team{
		ID:            team.ID,
		Name:          team.Name,
		ProductAreaID: team.ProductAreaID,
	}, nil
}

func (t *Teamkatalogen) GetTeamCatalogURL(teamID string) string {
	return t.url + "/team/" + teamID
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Nav-Consumer-Id", "nada-backend")
}
