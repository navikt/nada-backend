package story

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

type Handler struct {
	repo *database.Repo
}

func NewHandler(repo *database.Repo) *Handler {
	return &Handler{
		repo: repo,
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	authHeader := strings.Split(r.Header.Get("Authorization"), " ")
	if strings.ToLower(authHeader[0]) != "bearer" {
		http.Error(w, "Missing Bearer type", http.StatusForbidden)
		return
	}
	token := authHeader[1]

	tokenUID, err := uuid.Parse(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing, err := h.repo.GetStoryFromToken(r.Context(), tokenUID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newStory := &models.DBStory{}

	if err := json.NewDecoder(r.Body).Decode(newStory); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	draftID, err := h.repo.CreateStoryDraft(r.Context(), newStory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.repo.UpdateStory(r.Context(), models.NewStory{
		ID:               draftID,
		Name:             newStory.Name,
		Target:           &existing.ID,
		Group:            existing.Owner.Group,
		Keywords:         existing.Keywords,
		TeamkatalogenURL: existing.Owner.TeamkatalogenURL,
		TeamID:           existing.Owner.TeamID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	host := "http://localhost:3000"
	if os.Getenv("NAIS_CLUSTER_NAME") == "dev-gcp" {
		host = "https://data.intern.dev.nav.no"
	} else if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
		host = "https://data.intern.nav.no"
	}

	resp := map[string]string{
		"url": host + "/story/" + existing.ID.String(),
		"id":  existing.ID.String(),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
