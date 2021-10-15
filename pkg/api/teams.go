package api

import (
	"encoding/json"
	"net/http"

	"github.com/navikt/nada-backend/pkg/auth"
)

func (s *Server) GetGCPProjects(w http.ResponseWriter, r *http.Request, teamID string) {
	user := auth.GetUser(r.Context())

	found := false
	for _, t := range user.Groups {
		if t.Name == teamID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "No access", http.StatusUnauthorized)
		return
	}

	ps, ok := s.projectsMapping.Get(teamID)
	if !ok {
		http.Error(w, "No projects", http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ps); err != nil {
		s.log.WithError(err).Error("Encoding gcp projects as JSON")
	}
}
