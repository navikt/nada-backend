package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/httpwithcache"
	"github.com/navikt/nada-backend/pkg/service"
	"net/http"
	"strings"
)

var _ service.TeamKatalogenAPI = &teamKatalogenAPI{}

type teamKatalogenAPI struct {
	client *http.Client
	url    string
}

func (t *teamKatalogenAPI) GetProductArea(ctx context.Context, paID string) (*service.TeamkatalogenProductArea, error) {
	// TODO implement me
	panic("implement me")
}

func (t *teamKatalogenAPI) GetProductAreas(ctx context.Context) ([]*service.TeamkatalogenProductArea, error) {
	const op errs.Op = "teamKatalogenAPI.GetProductAreas"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/productarea", nil)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	q := req.URL.Query()
	q.Add("status", "active")
	req.URL.RawQuery = q.Encode()

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	var pasdto struct {
		Content []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			AreaType string `json:"areaType"`
		} `json:"content"`
	}

	if err := json.Unmarshal(res, &pasdto); err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	pas := make([]*service.TeamkatalogenProductArea, 0)
	for _, pa := range pasdto.Content {
		pas = append(pas, &service.TeamkatalogenProductArea{
			ID:       pa.ID,
			Name:     pa.Name,
			AreaType: pa.AreaType,
		})
	}

	return pas, nil
}

func (t *teamKatalogenAPI) GetTeam(ctx context.Context, teamID string) (*service.TeamkatalogenTeam, error) {
	const op errs.Op = "teamKatalogenAPI.GetTeam"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/team/"+teamID, nil)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	var team struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ProductAreaID string `json:"productAreaId"`
	}

	if err := json.Unmarshal(res, &team); err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	return &service.TeamkatalogenTeam{
		ID:            team.ID,
		Name:          team.Name,
		ProductAreaID: team.ProductAreaID,
	}, nil
}

func (t *teamKatalogenAPI) GetTeamCatalogURL(teamID string) string {
	return t.url + "/team/" + teamID
}

func (t *teamKatalogenAPI) GetTeamsInProductArea(ctx context.Context, paID string) ([]*service.TeamkatalogenTeam, error) {
	const op errs.Op = "teamKatalogenAPI.GetTeamsInProductArea"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url+"/team?status=ACTIVE&productAreaId="+paID, nil)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	var teams struct {
		Content []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ProductAreaID string `json:"productAreaId"`
		} `json:"content"`
	}

	if err := json.Unmarshal(res, &teams); err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	teamsGraph := make([]*service.TeamkatalogenTeam, len(teams.Content))
	for idx, t := range teams.Content {
		teamsGraph[idx] = &service.TeamkatalogenTeam{
			ID:            t.ID,
			Name:          t.Name,
			ProductAreaID: t.ProductAreaID,
		}
	}

	return teamsGraph, nil
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

func (t *teamKatalogenAPI) Search(ctx context.Context, gcpGroups []string) ([]service.TeamkatalogenResult, error) {
	const op errs.Op = "teamKatalogenAPI.Search"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/team?status=ACTIVE", t.url), nil)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	setRequestHeaders(req)
	res, err := httpwithcache.Do(t.client, req)
	if err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	var tkRes TeamkatalogenResponse
	if err := json.NewDecoder(bytes.NewReader(res)).Decode(&tkRes); err != nil {
		return nil, errs.E(errs.IO, op, err)
	}

	var ret []service.TeamkatalogenResult
	for _, r := range tkRes.Content {
		isMatch := false
		if ContainsAnyCaseInsensitive(r.Name, gcpGroups) {
			isMatch = true
		}
		for _, team := range r.NaisTeams {
			if ContainsAnyCaseInsensitive(team, gcpGroups) {
				isMatch = true
				break
			}
		}

		if isMatch {
			ret = append(ret, service.TeamkatalogenResult{
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

func ContainsAnyCaseInsensitive(s string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, q := range patterns {
		if strings.Contains(strings.ToLower(s), strings.ToLower(q)) {
			return true
		}
	}
	return false
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Nav-Consumer-Id", "nada-backend")
}

func NewTeamKatalogenAPI(url string) *teamKatalogenAPI {
	return &teamKatalogenAPI{
		// FIXME: inject
		client: http.DefaultClient,
		url:    url,
	}
}
