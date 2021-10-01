package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"net/http"
)

func UpdateTeams(c context.Context, mapToUpdate map[string]string, teamsURL, teamsToken string, updateFrequency time.Duration) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fetchTeams(c, mapToUpdate, teamsURL, teamsToken); err != nil {
				log.Errorf("Fetching teams from url: %v: %v", teamsURL, err)
			}
			ticker.Reset(updateFrequency)
		case <-c.Done():
			return
		}
	}
}

func fetchTeams(c context.Context, mapToUpdate map[string]string, teamsURL, teamsToken string) error {
	request, err := http.NewRequestWithContext(c, http.MethodGet, teamsURL, nil)
	if err != nil {
		log.Errorf("Creating http request for teams: %v", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", teamsToken))
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("performing http request on teams json URL: %v: %w", teamsURL, err)
	}

	if err := json.NewDecoder(response.Body).Decode(&mapToUpdate); err != nil {
		return fmt.Errorf("unmarshalling response from teams json URL: %v: %w", teamsURL, err)
	}

	log.Infof("Updated UUID mapping: %d teams from %v", len(mapToUpdate), teamsURL)
	return nil
}
