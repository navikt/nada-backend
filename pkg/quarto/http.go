package quarto

import (
	"net/http"
	"strings"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
)

type Handler struct {
	repo       *database.Repo
	gcsClient  gcs.GCS
	bucketName string
}

func NewHandler(repo *database.Repo, gcsClient gcs.GCS) *Handler {
	return &Handler{
		repo:      repo,
		gcsClient: gcsClient,
	}
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	qID := pathParts[2]

	objPath, err := h.gcsClient.GetIndexHtmlPath(r.Context(), qID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	http.Redirect(w, r, objPath, http.StatusSeeOther)
}

func (h *Handler) GetObject(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimLeft(r.URL.Path, "/quarto")

	objBytes, err := h.gcsClient.GetObject(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Add("content-type", "text/html")
	case strings.HasSuffix(path, ".css"):
		w.Header().Add("content-type", "text/css")
	case strings.HasSuffix(path, ".js"):
		w.Header().Add("content-type", "application/javascript")
	case strings.HasSuffix(path, ".json"):
		w.Header().Add("content-type", "application/json")
	}

	w.Write(objBytes)
}
