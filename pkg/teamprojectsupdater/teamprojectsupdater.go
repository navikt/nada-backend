package teamprojectsupdater

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/leaderelection"
	log "github.com/sirupsen/logrus"
)

type TeamProjectsUpdater struct {
	TeamProjectsMapping *auth.TeamProjectsMapping
	consoleURL          string
	consoleAPIKey       string
	httpClient          *http.Client
	repo                *database.Repo
}

type Team struct {
	GoogleGroupsEmail string `json:"googleGroupEmail"`
	Environments      []struct {
		Name         string `json:"name"`
		GcpProjectID string `json:"gcpProjectID"`
	} `json:"environments"`
}

func NewTeamProjectsUpdater(ctx context.Context, consoleURL, consoleAPIKey string, httpClient *http.Client, repo *database.Repo) *TeamProjectsUpdater {
	teamProjectsSQL, err := repo.GetTeamProjects(ctx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.WithError(err).Errorf("Fetching teams from database")
		}
	}

	teamprojects := map[string]string{}
	for _, project := range teamProjectsSQL {
		teamprojects[project.Team] = project.Project
	}

	teamProjectsMapping := &auth.TeamProjectsMapping{}
	teamProjectsMapping.SetTeamProjects(teamprojects)

	return &TeamProjectsUpdater{
		TeamProjectsMapping: teamProjectsMapping,
		consoleURL:          consoleURL,
		consoleAPIKey:       consoleAPIKey,
		httpClient:          httpClient,
		repo:                repo,
	}
}

func NewMockTeamProjectsUpdater(ctx context.Context, repo *database.Repo) (*TeamProjectsUpdater, error) {
	tpu := &TeamProjectsUpdater{
		TeamProjectsMapping: &auth.TeamProjectsMapping{
			TeamProjects: map[string]string{
				"team@nav.no": "team-dev-1337",
				"nada@nav.no": "dataplattform-dev-9da3",
				"aura@nav.no": "aura-dev-d9f5",
			},
		},
	}

	if err := repo.UpdateTeamProjectsCache(ctx, map[string]string{
		"team": "team-dev-1337",
		"nada": "dataplattform-dev-9da3",
		"aura": "aura-dev-d9f5",
	}); err != nil {
		return nil, err
	}

	return tpu, nil
}

func (t *TeamProjectsUpdater) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	time.Sleep(time.Second * 60)

	for {
		if err := t.FetchTeamGoogleProjectsMapping(ctx); err != nil {
			log.WithError(err).Errorf("Fetching teams")
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (t *TeamProjectsUpdater) fetchTeamGoogleProjects(ctx context.Context, limit, offset int) (teams []Team, hasMore bool, err error) {
	type response struct {
		Data struct {
			Teams struct {
				Nodes    []Team `json:"nodes"`
				PageInfo struct {
					HasNextPage bool `json:"hasNextPage"`
				}
			} `json:"teams"`
		} `json:"data"`
	}

	gqlQuery := `
		query GCPTeams($limit: Int, $offset: Int){
			teams(limit: $limit, offset: $offset) {
				nodes {
					googleGroupEmail
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

	payload := map[string]any{"query": gqlQuery, "variables": map[string]any{"limit": limit, "offset": offset}}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, false, err
	}

	r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/query", t.consoleURL), bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, false, err
	}
	r.Header.Add("authorization", fmt.Sprintf("Bearer %v", t.consoleAPIKey))
	r.Header.Add("content-type", "application/json")

	res, err := t.httpClient.Do(r)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()

	data := response{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, false, err
	}

	return data.Data.Teams.Nodes, data.Data.Teams.PageInfo.HasNextPage, nil
}

func (t *TeamProjectsUpdater) FetchTeamGoogleProjectsMapping(ctx context.Context) error {
	all := []Team{}

	const limit = 100
	offset := 0
	for {
		res, more, err := t.fetchTeamGoogleProjects(ctx, limit, offset)
		if err != nil {
			return err
		}

		all = append(all, res...)

		if !more {
			break
		}

		offset += limit
	}

	projectMapping, err := t.getTeamProjectsMappingForEnv(all)
	if err != nil {
		return err
	}

	if len(projectMapping) == 0 {
		return fmt.Errorf("error parsing team projects from console, %v teams was mapped to %v team projects", len(all), len(projectMapping))
	}

	t.TeamProjectsMapping.SetTeamProjects(projectMapping)

	isLeader, err := leaderelection.IsLeader()
	if err != nil {
		return err
	}

	if isLeader {
		if err := t.repo.UpdateTeamProjectsCache(ctx, projectMapping); err != nil {
			return err
		}
	}

	return nil
}

func (t *TeamProjectsUpdater) getTeamProjectsMappingForEnv(teams []Team) (map[string]string, error) {
	env := os.Getenv("NAIS_CLUSTER_NAME")
	if env == "" {
		env = "dev-gcp"
	}

	out := map[string]string{}
	for _, t := range teams {
		for _, p := range t.Environments {
			if p.Name == env {
				parts := strings.Split(t.GoogleGroupsEmail, "@")
				if len(parts) != 2 {
					return nil, fmt.Errorf("incorrect email format for group %v", t.GoogleGroupsEmail)
				}
				out[parts[0]] = p.GcpProjectID
			}
		}
	}

	return out, nil
}
