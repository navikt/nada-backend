package auth

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

type TeamProjectsMapping struct {
	lock         sync.RWMutex
	TeamProjects map[string]string
}

func (t *TeamProjectsMapping) Get(team string) (string, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	project, ok := t.TeamProjects[team]
	return project, ok
}

func (t *TeamProjectsMapping) SetTeamProjects(projects map[string]string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.TeamProjects = map[string]string{}
	mergeInto(t.TeamProjects, projects)
	log.Infof("Updated team projects mapping: %v teams", len(t.TeamProjects))
}

func mergeInto(result map[string]string, projects map[string]string) {
	for key, value := range projects {
		result[key+"@nav.no"] = value
	}
}
