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

type TeamsCache struct {
	url   string
	token string

	lock sync.RWMutex
	data map[string]string
}

func NewTeamsCache(url, token string) *TeamsCache {
	return &TeamsCache{
		url:   url,
		token: token,
		data:  map[string]string{},
	}
}

func (t *TeamsCache) Run(ctx context.Context, updateFrequency time.Duration) {
	ticker := time.NewTicker(updateFrequency)
	defer ticker.Stop()

	for {
		if err := t.fetchTeams(ctx); err != nil {
			log.Errorf("Fetching teams from url: %v: %v", t.url, err)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (t *TeamsCache) Get(uuid string) (string, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	name, ok := t.data[uuid]
	return name, ok
}

func (t *TeamsCache) fetchTeams(c context.Context) error {
	request, err := http.NewRequestWithContext(c, http.MethodGet, t.url, nil)
	if err != nil {
		log.Errorf("Creating http request for teams: %v", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", t.token))
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("performing http request on teams json URL: %v: %w", t.url, err)
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	if err := json.NewDecoder(response.Body).Decode(&t.data); err != nil {
		return fmt.Errorf("unmarshalling response from teams json URL: %v: %w", t.url, err)
	}

	log.Infof("Updated UUID mapping: %d teams from %v", len(t.data), t.url)
	return nil
}
