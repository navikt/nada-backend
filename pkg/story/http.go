package story

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
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

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	story := &models.DBStory{}

	if err := json.NewDecoder(r.Body).Decode(story); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.repo.CreateStoryDraft(r.Context(), story)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Add("content-type", "application/json")

	resp := map[string]string{
		"url": r.Host + "/story/draft/" + id.String(),
		"id":  id.String(),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Println(err)
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	storyID := chi.URLParam(r, "id")

	uid, err := uuid.Parse(storyID)
	if err != nil {
		fmt.Println("from bytes")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storyToken, err := h.repo.GetStoryToken(r.Context(), uid)
	if err != nil {
		fmt.Println("get token")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if token != storyToken {
		fmt.Println("token unauthorized")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	story := &models.DBStory{}

	if err := json.NewDecoder(r.Body).Decode(story); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	draftID, err := h.repo.CreateStoryDraft(r.Context(), story)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.repo.UpdateStory(r.Context(), draftID, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := map[string]string{
		"url": r.Host + "/story/" + storyID,
		"id":  storyID,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
