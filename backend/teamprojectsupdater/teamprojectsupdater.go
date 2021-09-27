package teamprojectsupdater

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/navikt/datakatalogen/backend/config"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type OutputFile struct {
	TeamProjectIdMapping OutputVariable `json:"team_projectid_mapping"`
}

type OutputVariable struct {
	Value []map[string]string `json:"value"`
}

type TeamProjectsUpdater struct {
	ctx             context.Context
	teamProjects    map[string][]string
	cfg             config.Config
	updateFrequency time.Duration
	httpClient      *http.Client
}

func New(c context.Context, teamProjects map[string][]string, config config.Config, updateFrequency time.Duration, httpClient *http.Client) *TeamProjectsUpdater {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &TeamProjectsUpdater{
		ctx:             c,
		teamProjects:    teamProjects,
		cfg:             config,
		updateFrequency: updateFrequency,
		httpClient:      httpClient,
	}
}

func (t *TeamProjectsUpdater) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := FetchTeamGoogleProjectsMapping(t.ctx, t.teamProjects, t.cfg); err != nil {
				log.Errorf("Fetching teams from url: %v: %v", t.cfg.ProdTeamProjectsOutputURL, err)
			}

			log.Infof("Updated team GCP projects map for %v teams", len(t.teamProjects))
			ticker.Reset(t.updateFrequency)
		case <-t.ctx.Done():
			return
		}
	}
}

func FetchTeamGoogleProjectsMapping(c context.Context, teamProjects map[string][]string, config config.Config) error {
	devOutputFile, err := getOutputFile(c, config.DevTeamProjectsOutputURL, config.TeamsToken)
	if err != nil {
		return err
	}
	prodOutputFile, err := getOutputFile(c, config.ProdTeamProjectsOutputURL, config.TeamsToken)
	if err != nil {
		return err
	}

	mergeInto(teamProjects, devOutputFile.TeamProjectIdMapping.Value, prodOutputFile.TeamProjectIdMapping.Value)

	return nil
}

func getOutputFile(ctx context.Context, url, token string) (*OutputFile, error) {
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

	if err := json.NewDecoder(response.Body).Decode(&outputFile); err != nil {
		return nil, fmt.Errorf("unmarshalling terraform output file: %w", err)
	}

	return &outputFile, nil
}

func mergeInto(result map[string][]string, first []map[string]string, second []map[string]string) {
	for _, item := range first {
		for key, value := range item {
			result[key] = append(result[key], value)
		}
	}
	for _, item := range second {
		for key, value := range item {
			result[key] = append(result[key], value)
		}
	}
}
