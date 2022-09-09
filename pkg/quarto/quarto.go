package quarto

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/navikt/nada-backend/pkg/database"
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
	var quarto struct {
		Content string
	}

	if err := json.NewDecoder(r.Body).Decode(&quarto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.repo.CreateQuarto(r.Context(), "nada@nav.no", quarto.Content)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	host := "http://localhost:3000"
	if os.Getenv("NAIS_CLUSTER_NAME") == "dev-gcp" {
		host = "https://data.dev.intern.nav.no"
	} else if os.Getenv("NAIS_CLUSTER_NAME") == "prod-gcp" {
		host = "https://data.intern.nav.no"
	}

	w.Header().Add("content-type", "application/json")

	resp := map[string]string{
		"id":  id.String(),
		"url": host + "/quarto/" + id.String(),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Println(err)
	}
}