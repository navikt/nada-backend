package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type TeamProjectsUpdater struct {
	lock                sync.RWMutex
	teamProjects        map[string][]string
	devTeamProjectsURL  string
	prodTeamProjectsURL string
	teamsToken          string
	httpClient          *http.Client
}

type OutputFile struct {
	TeamProjectIDMapping OutputVariable `json:"team_projectid_mapping"`
}

type OutputVariable struct {
	Value map[string]string `json:"value"`
}

func NewTeamProjectsUpdater(devTeamProjectsURL, prodTeamProjectsURL, teamsToken string, httpClient *http.Client) *TeamProjectsUpdater {
	return &TeamProjectsUpdater{
		teamProjects:        make(map[string][]string),
		devTeamProjectsURL:  devTeamProjectsURL,
		prodTeamProjectsURL: prodTeamProjectsURL,
		teamsToken:          teamsToken,
		httpClient:          httpClient,
	}
}

func (t *TeamProjectsUpdater) Run(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

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

func (t *TeamProjectsUpdater) Get(team string) ([]string, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	projects, ok := t.teamProjects[team]
	return projects, ok
}

func (t *TeamProjectsUpdater) OwnsProject(team, project string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	projects, ok := t.teamProjects[team]
	if !ok {
		return false
	}
	return contains(project, projects)
}

func contains(elem string, list []string) bool {
	for _, entry := range list {
		if entry == elem {
			return true
		}
	}
	return false
}

func (t *TeamProjectsUpdater) FetchTeamGoogleProjectsMapping(ctx context.Context) error {
	devOutputFile, err := getOutputFile(ctx, t.devTeamProjectsURL, t.teamsToken)
	if err != nil {
		return err
	}
	prodOutputFile, err := getOutputFile(ctx, t.prodTeamProjectsURL, t.teamsToken)
	if err != nil {
		return err
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	t.teamProjects = map[string][]string{}
	mergeInto(t.teamProjects, devOutputFile, prodOutputFile)
	log.Infof("Updated team projects mapping: %v teams", len(t.teamProjects))

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

func mergeInto(result map[string][]string, first map[string]string, second map[string]string) {
	for key, value := range first {
		result[key+"@nav.no"] = append(result[key+"@nav.no"], value)
	}
	for key, value := range second {
		result[key+"@nav.no"] = append(result[key+"@nav.no"], value)
	}
}
