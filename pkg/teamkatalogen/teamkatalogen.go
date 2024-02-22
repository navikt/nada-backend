package teamkatalogen

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/httpwithcache"
	"github.com/sirupsen/logrus"
)

type TeamkatalogenResult struct {
	TeamID        string `json:"teamID"`
	URL           string `json:"url"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ProductAreaID string `json:"productAreaID"`
}

type ProductArea struct {
	// id is the id of the product area.
	ID string `json:"id"`
	// name is the name of the product area.
	Name string `json:"name"`
	//areaType is the type of the product area.
	AreaType string `json:"areaType"`
	Teams    []Team `json:"teams"`
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
	Search(ctx context.Context, gcpGroups []string) ([]TeamkatalogenResult, error)
	GetTeamsInProductArea(ctx context.Context, paID string) ([]*Team, error)
	GetProductArea(ctx context.Context, paID string) (*ProductArea, error)
	GetProductAreas(ctx context.Context) ([]*ProductArea, error)
	GetTeam(ctx context.Context, teamID string) (*Team, error)
	GetTeamCatalogURL(teamID string) string
}

type teamkatalogen struct {
	client  *http.Client
	url     string
	log     *logrus.Logger
	querier gensql.Querier
	db      *sql.DB
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

func New(url string, db *sql.DB, querier gensql.Querier, log *logrus.Logger) Teamkatalogen {
	tk := &teamkatalogen{client: http.DefaultClient, url: url, log: log, db: db, querier: querier}
	tk.RunSyncer()
	return tk
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

func (t *teamkatalogen) Search(ctx context.Context, gcpGroups []string) ([]TeamkatalogenResult, error) {
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

	ret := []TeamkatalogenResult{}
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
			ret = append(ret, TeamkatalogenResult{
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

func (t *teamkatalogen) restGetProductAreas(ctx context.Context) ([]*ProductArea, error) {
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

func (t *teamkatalogen) GetProductAreas(ctx context.Context) ([]*ProductArea, error) {
	pas, err := t.querier.GetProductAreas(ctx)
	if err != nil {
		return nil, err
	}
	teams, err := t.querier.GetAllTeams(ctx)
	if err != nil {
		return nil, err
	}

	productAreas := make([]*ProductArea, 0)
	for _, pa := range pas {
		paTeams := make([]Team, 0)
		for _, team := range teams {
			if team.ProductAreaID.Valid && team.ProductAreaID.UUID == pa.ID {
				paTeams = append(paTeams, Team{
					ID:            team.ID.String(),
					Name:          team.Name.String,
					ProductAreaID: team.ProductAreaID.UUID.String(),
				})
			}
		}
		areaType := ""
		if pa.AreaType.Valid {
			areaType = pa.AreaType.String
		}
		productAreas = append(productAreas, &ProductArea{
			ID:       pa.ID.String(),
			Name:     pa.Name.String,
			AreaType: areaType,
			Teams:    paTeams,
		})
	}

	return productAreas, nil
}

func (t *teamkatalogen) GetProductArea(ctx context.Context, paID string) (*ProductArea, error) {
	pa, err := t.querier.GetProductArea(ctx, uuid.MustParse(paID))
	if err != nil {
		return nil, err
	}
	teams, err := t.querier.GetTeamsInProductArea(ctx, uuid.NullUUID{
		UUID:  pa.ID,
		Valid: true,
	})
	if err != nil {
		return nil, err
	}

	paTeams := make([]Team, 0)
	for _, team := range teams {
		paTeams = append(paTeams, Team{
			ID:            team.ID.String(),
			Name:          team.Name.String,
			ProductAreaID: team.ProductAreaID.UUID.String(),
		})
	}

	areaType := ""
	if pa.AreaType.Valid {
		areaType = pa.AreaType.String
	}
	return &ProductArea{
		ID:       pa.ID.String(),
		Name:     pa.Name.String,
		AreaType: areaType,
		Teams:    paTeams,
	}, nil
}
