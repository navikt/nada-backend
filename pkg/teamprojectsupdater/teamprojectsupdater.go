package teamprojectsupdater

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	ReconcilerState struct {
		GoogleWorkspaceGroupEmail string `json:"googleWorkspaceGroupEmail"`
		GCPProjects               []struct {
			Environment string `json:"environment"`
			ProjectID   string `json:"projectId"`
		} `json:"gcpProjects"`
	} `json:"reconcilerState"`
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

func (t *TeamProjectsUpdater) FetchTeamGoogleProjectsMapping(ctx context.Context) error {
	type response struct {
		Data struct {
			Teams []Team `json:"teams"`
		} `json:"data"`
	}

	gqlQuery := `
	{
		teams {
		  reconcilerState {
			  googleWorkspaceGroupEmail
			  gcpProjects {
				  environment
				  projectId
			  }
		  }
		}
	}
	`

	payload := map[string]string{"query": gqlQuery}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/query", t.consoleURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	r.Header.Add("authorization", fmt.Sprintf("Bearer %v", t.consoleAPIKey))
	r.Header.Add("content-type", "application/json")

	res, err := t.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	data := response{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return err
	}

	projectMapping, err := t.getTeamProjectsMappingForEnv(data.Data.Teams)
	if err != nil {
		return err
	}

	if len(projectMapping) == 0 {
		return fmt.Errorf("error parsing team projects from console, %v teams was mapped to %v team projects", len(data.Data.Teams), len(projectMapping))
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
		for _, p := range t.ReconcilerState.GCPProjects {
			if p.Environment == env {
				parts := strings.Split(t.ReconcilerState.GoogleWorkspaceGroupEmail, "@")
				if len(parts) != 2 {
					return nil, fmt.Errorf("incorrect email format for group %v", t.ReconcilerState.GoogleWorkspaceGroupEmail)
				}
				out[parts[0]] = p.ProjectID
			}
		}
	}

	return out, nil
}
