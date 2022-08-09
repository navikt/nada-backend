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
	"time"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/leaderelection"
	log "github.com/sirupsen/logrus"
)

type TeamProjectsUpdater struct {
	TeamProjectsMapping *auth.TeamProjectsMapping
	teamProjectsURL     string
	teamsToken          string
	httpClient          *http.Client
	repo                *database.Repo
}

type OutputFile struct {
	TeamProjectIDMapping OutputVariable `json:"team_projectid_mapping"`
}

type OutputVariable struct {
	Value map[string]string `json:"value"`
}

func NewTeamProjectsUpdater(teamProjectsURL, teamsToken string, httpClient *http.Client, repo *database.Repo) *TeamProjectsUpdater {
	teamProjectsMapping := &auth.TeamProjectsMapping{}
	return &TeamProjectsUpdater{
		TeamProjectsMapping: teamProjectsMapping,
		teamProjectsURL:     teamProjectsURL,
		teamsToken:          teamsToken,
		httpClient:          httpClient,
		repo:                repo,
	}
}

func NewMockTeamProjectsUpdater() *TeamProjectsUpdater {
	return &TeamProjectsUpdater{
		TeamProjectsMapping: &auth.TeamProjectsMapping{
			TeamProjects: map[string]string{
				"team@nav.no": "team-dev-1337",
				"nada@nav.no": "dataplattform-dev-9da3",
				"aura@nav.no": "aura-dev-d9f5",
			},
		},
	}
}

func (t *TeamProjectsUpdater) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	teamProjectsSQL, err := t.repo.GetTeamProjects(ctx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.WithError(err).Errorf("Fetching teams from database")
		}
	}

	teamprojects := map[string]string{}
	for _, project := range teamProjectsSQL {
		teamprojects[project.Team] = project.Project
	}

	t.TeamProjectsMapping.SetTeamProjects(teamprojects)

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
	outputFile, err := getOutputFile(ctx, t.teamProjectsURL, t.teamsToken)
	if err != nil {
		return err
	}

	t.TeamProjectsMapping.SetTeamProjects(outputFile)

	isLeader, err := leaderelection.IsLeader()
	if err != nil {
		return err
	}

	if isLeader {
		if err := t.repo.UpdateTeamProjectsCache(ctx, outputFile); err != nil {
			return err
		}
	}

	return nil
}

func getOutputFile(ctx context.Context, url, token string) (map[string]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request for getting terraform output file: %w", err)
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("performing http request, URL: %v: %w", url, err)
	}

	var outputFile OutputFile

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	woFirstLine := bytes.Split(bodyBytes, []byte("-json"))
	if len(woFirstLine) != 1 {
		// Remove terraform output from file
		body := bytes.Split(woFirstLine[1], []byte("::debug"))
		if err := json.Unmarshal(body[0], &outputFile); err != nil {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(bodyBytes, &outputFile); err != nil {
			return nil, err
		}
	}

	return outputFile.TeamProjectIDMapping.Value, nil
}
