package auth

import (
	"context"
	"encoding/json"
	"fmt"
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

	var outputFile map[string]string

	if err := json.NewDecoder(response.Body).Decode(&outputFile); err != nil {
		return nil, fmt.Errorf("unmarshalling terraform output file: %w", err)
	}

	return outputFile, nil
}

func mergeInto(result map[string][]string, first map[string]string, second map[string]string) {
	for key, value := range first {
		result[key+"@nav.no"] = append(result[key+"@nav.no"], value)
	}
	for key, value := range second {
		result[key+"@nav.no"] = append(result[key+"@nav.no"], value)
	}
}
