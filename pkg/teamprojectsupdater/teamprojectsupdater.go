package teamprojectsupdater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type OutputFile struct {
	TeamProjectIdMapping OutputVariable `json:"team_projectid_mapping"`
}

type OutputVariable struct {
	Value []map[string]string `json:"value"`
}

type TeamProjectsUpdater struct {
	ctx                 context.Context
	teamProjects        map[string][]string
	devTeamProjectsURL  string
	prodTeamProjectsURL string
	teamsToken          string
	updateFrequency     time.Duration
	httpClient          *http.Client
}

func New(c context.Context, teamProjects map[string][]string, devTeamProjectsURL, prodTeamProjectsURL, teamsToken string, updateFrequency time.Duration, httpClient *http.Client) *TeamProjectsUpdater {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &TeamProjectsUpdater{
		ctx:                 c,
		teamProjects:        teamProjects,
		devTeamProjectsURL:  devTeamProjectsURL,
		prodTeamProjectsURL: prodTeamProjectsURL,
		teamsToken:          teamsToken,
		updateFrequency:     updateFrequency,
		httpClient:          httpClient,
	}
}

func (t *TeamProjectsUpdater) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := t.FetchTeamGoogleProjectsMapping(); err != nil {
				log.WithError(err).Errorf("Fetching teams")
			}

			log.Infof("Updated team GCP projects map for %v teams", len(t.teamProjects))
			ticker.Reset(t.updateFrequency)
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *TeamProjectsUpdater) FetchTeamGoogleProjectsMapping() error {
	devOutputFile, err := getOutputFile(t.ctx, t.devTeamProjectsURL, t.teamsToken)
	if err != nil {
		return err
	}
	prodOutputFile, err := getOutputFile(t.ctx, t.prodTeamProjectsURL, t.teamsToken)
	if err != nil {
		return err
	}

	mergeInto(t.teamProjects, devOutputFile.TeamProjectIdMapping.Value, prodOutputFile.TeamProjectIdMapping.Value)

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
