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
	"time"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/leaderelection"
	log "github.com/sirupsen/logrus"
)

func GetTeamProjects(ctx context.Context, repo *database.Repo) ([]gensql.TeamProject, error) {
	teamProjects, err := repo.Querier.GetTeamProjects(ctx)
	if err != nil {
		return nil, err
	}
	return teamProjects, nil
}

func UpdateTeamProjectsCache(ctx context.Context, repo *database.Repo, teamProjects map[string]string) error {
	tx, err := repo.GetDB().Begin()
	if err != nil {
		return err
	}

	querier := repo.Querier.WithTx(tx)

	if err := querier.ClearTeamProjectsCache(ctx); err != nil {
		if err := tx.Rollback(); err != nil {
			log.WithError(err).Error("Rolling back clear projects cache transaction")
		}
		return err
	}

	for team, projectID := range teamProjects {
		_, err := querier.AddTeamProject(ctx, gensql.AddTeamProjectParams{
			Team:    team,
			Project: projectID,
		})
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.WithError(err).Error("Rolling back update projects cache transaction")
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

type TeamProjectsUpdater struct {
	TeamProjectsMapping *auth.TeamProjectsMapping
	consoleURL          string
	consoleAPIKey       string
	httpClient          *http.Client
	repo                *database.Repo
}

type Team struct {
	Slug         string `json:"slug"`
	Environments []struct {
		Name         string `json:"name"`
		GcpProjectID string `json:"gcpProjectID"`
	} `json:"environments"`
}

func NewTeamProjectsUpdater(ctx context.Context, consoleURL, consoleAPIKey string, httpClient *http.Client, repo *database.Repo) *TeamProjectsUpdater {
	teamProjectsSQL, err := GetTeamProjects(ctx, repo)
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
				"team@nav.no":   "team-dev-1337",
				"nada@nav.no":   "dataplattform-dev-9da3",
				"aura@nav.no":   "aura-dev-d9f5",
				"nyteam@nav.no": "nyteam-dev-1234",
			},
		},
	}

	if err := UpdateTeamProjectsCache(ctx, repo, map[string]string{
		"team":   "team-dev-1337",
		"nada":   "dataplattform-dev-9da3",
		"aura":   "aura-dev-d9f5",
		"nyteam": "nyteam-dev-1234",
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
				} `json:"pageInfo"`
			} `json:"teams"`
		} `json:"data"`
	}

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
		if err := UpdateTeamProjectsCache(ctx, t.repo, projectMapping); err != nil {
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
				out[t.Slug] = p.GcpProjectID
			}
		}
	}

	return out, nil
}
