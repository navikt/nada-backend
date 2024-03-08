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

type ProductArea struct {
	// id is the id of the product area.
	ID string `json:"id"`
	// name is the name of the product area.
	Name string `json:"name"`
	//areaType is the type of the product area.
	AreaType string `json:"areaType"`
}

type Team struct {
	// id is the team external id in teamkatalogen.
	ID string `json:"id"`
	// name is the name of the team.
	Name string `json:"name"`
	// productAreaID is the id of the product area.
	ProductAreaID string `json:"productAreaID"`
}

type Teamkatalogen interface {
	Search(ctx context.Context, query string) ([]*models.TeamkatalogenResult, error)
	GetTeamsInProductArea(ctx context.Context, paID string) ([]*Team, error)
	GetProductArea(ctx context.Context, paID string) (*ProductArea, error)
	GetProductAreas(ctx context.Context) ([]*ProductArea, error)
	GetTeam(ctx context.Context, teamID string) (*Team, error)
	GetTeamCatalogURL(teamID string) string
}

type teamkatalogen struct {
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

func New(url string) Teamkatalogen {
	return &teamkatalogen{client: http.DefaultClient, url: url}
}

func (t *teamkatalogen) Search(ctx context.Context, query string) ([]*models.TeamkatalogenResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/team?status=ACTIVE", t.url), nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve teams from team catalogue")
	}

	var tkRes TeamkatalogenResponse
	if err := json.NewDecoder(bytes.NewReader(res)).Decode(&tkRes); err != nil {
		return nil, fmt.Errorf("unable to retrieve teams from team catalogue")
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

func (t *teamkatalogen) GetTeamsInProductArea(ctx context.Context, paID string) ([]*Team, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/team?status=ACTIVE&productAreaId="+paID, nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve teams in product area with id '%v' from team catalogue", paID)
	}

	var teams struct {
		Content []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ProductAreaID string `json:"productAreaId"`
		} `json:"content"`
	}

	if err := json.Unmarshal(res, &teams); err != nil {
		return nil, fmt.Errorf("unable to retrieve teams in product area with id '%v' from team catalogue", paID)
	}

	teamsGraph := make([]*Team, len(teams.Content))
	for idx, t := range teams.Content {
		teamsGraph[idx] = &Team{
			ID:            t.ID,
			Name:          t.Name,
			ProductAreaID: t.ProductAreaID,
		}
	}

	return teamsGraph, nil
}

func (t *teamkatalogen) GetProductArea(ctx context.Context, paID string) (*ProductArea, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/productarea/"+paID, nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, fmt.Errorf("unable to get product area '%v' from team catalogue", paID)
	}

	var pa struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(res, &pa); err != nil {
		return nil, fmt.Errorf("unable to get product area '%v' from team catalogue", paID)
	}
	return &ProductArea{
		ID:   pa.ID,
		Name: pa.Name,
	}, nil
}

func (t *teamkatalogen) GetProductAreas(ctx context.Context) ([]*ProductArea, error) {
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
		return nil, fmt.Errorf("unable to retrieve product areas from team catalogue")
	}

	var pasdto struct {
		Content []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			AreaType string `json:"areaType"`
		} `json:"content"`
	}

	if err := json.Unmarshal(res, &pasdto); err != nil {
		return nil, fmt.Errorf("unable to retrieve product areas from team catalogue")
	}

	pas := make([]*ProductArea, 0)
	for _, pa := range pasdto.Content {
		pas = append(pas, &ProductArea{
			ID:       pa.ID,
			Name:     pa.Name,
			AreaType: pa.AreaType,
		})
	}

	return pas, nil
}

func (t *teamkatalogen) GetTeam(ctx context.Context, teamID string) (*Team, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/team/"+teamID, nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve team '%v' from team catalogue", teamID)
	}

	var team struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ProductAreaID string `json:"productAreaId"`
	}

	if err := json.Unmarshal(res, &team); err != nil {
		return nil, fmt.Errorf("unable to retrieve team '%v' from team catalogue", teamID)
	}

	return &Team{
		ID:            team.ID,
		Name:          team.Name,
		ProductAreaID: team.ProductAreaID,
	}, nil
}

func (t *teamkatalogen) GetTeamCatalogURL(teamID string) string {
	return t.url + "/team/" + teamID
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Nav-Consumer-Id", "nada-backend")
}
