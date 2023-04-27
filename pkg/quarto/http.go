package quarto

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/sirupsen/logrus"
)

const (
	idURLPosUpdate         = 3
	idURLPosGet            = 2
	maxMemoryMultipartForm = 32 << 20 // 32 MB
)

type Handler struct {
	repo      *database.Repo
	gcsClient *gcs.Client
	log       *logrus.Entry
}

func NewHandler(repo *database.Repo, gcsClient *gcs.Client, logger *logrus.Entry) *Handler {
	return &Handler{
		repo:      repo,
		gcsClient: gcsClient,
		log:       logger,
	}
}

func (h *Handler) GetObject(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/quarto/")

	attr, objBytes, err := h.gcsClient.GetObject(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("content-type", attr.ContentType)
	w.Header().Add("content-length", strconv.Itoa(int(attr.Size)))
	w.Header().Add("content-encoding", attr.ContentEncoding)

	w.Write(objBytes)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	qID, err := getIDFromPath(r, idURLPosUpdate)
	if err != nil {
		h.log.WithError(err).Errorf("getting quarto id from url path")
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid quarto id %v", qID))
		return
	}

	err = r.ParseMultipartForm(maxMemoryMultipartForm)
	if err != nil {
		h.log.WithError(err).Errorf("parsing multipart form")
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request form"))
		return
	}

	for _, fileHeader := range r.MultipartForm.File {
		if err := h.uploadFile(r.Context(), qID.String(), fileHeader); err != nil {
			h.log.WithError(err).Errorf("uploading file")
			h.writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			return
		}
	}
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	qID, err := getIDFromPath(r, idURLPosGet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	r.URL.Path = "/quarto/"

	objPath, err := h.gcsClient.GetIndexHtmlPath(r.Context(), qID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	http.Redirect(w, r, objPath, http.StatusSeeOther)
}

func (h *Handler) QuartoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.getQuarto(w, r, next)
		case http.MethodPut:
			h.updateQuarto(w, r, next)
		}
	})
}

func (h *Handler) getQuarto(w http.ResponseWriter, r *http.Request, next http.Handler) {
	regex, _ := regexp.Compile(`[\n]*\.[\n]*`) // check if object path has file extension
	if !regex.MatchString(r.URL.Path) {
		h.Redirect(w, r)
		return
	}

	next.ServeHTTP(w, r)
}

func (h *Handler) updateQuarto(w http.ResponseWriter, r *http.Request, next http.Handler) {
	qID, err := getIDFromPath(r, idURLPosUpdate)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid quarto id %v", qID))
		return
	}

	authHeader := r.Header.Get("Authorization")
	token, err := getTokenFromHeader(authHeader)
	if err != nil {
		h.writeError(w, http.StatusForbidden, err)
		return
	}

	story, err := h.repo.GetQuartoStory(r.Context(), qID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, fmt.Errorf("quarto id %v does not exist", qID))
			return
		}

		h.log.WithError(err).Errorf("reading quarto id %v", qID)
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	group := strings.Split(story.Group, "@")[0]
	dbToken, err := h.repo.GetNadaToken(r.Context(), group)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.log.Errorf("no nada token found for team %v, quarto id %v", story.Group, qID)
			h.writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			return
		}

		h.log.WithError(err).Errorf("reading nada token for group %v, quarto id %v", story.Group, qID)
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	if !isAuthorized(token, dbToken) {
		h.writeError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized to update quarto %v", qID))
		return
	}

	next.ServeHTTP(w, r)
}

func (h *Handler) uploadFile(ctx context.Context, objPath string, fileHeader []*multipart.FileHeader) error {
	for _, f := range fileHeader {
		file, err := f.Open()
		if err != nil {
			return err
		}
		if err := h.gcsClient.UploadFile(ctx, objPath, file); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) writeError(w http.ResponseWriter, status int, err error) {
	resp := map[string]string{
		"statusCode": strconv.Itoa(status),
		"message":    err.Error(),
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		h.log.WithError(err).Errorf("marshalling error response")
	}

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func getIDFromPath(r *http.Request, idPos int) (uuid.UUID, error) {
	parts := strings.Split(r.URL.Path, "/")
	if idPos > len(parts)-1 {
		return uuid.UUID{}, fmt.Errorf("unable to extract id from url path")
	}

	id, err := uuid.Parse(parts[idPos])
	if err != nil {
		return uuid.UUID{}, err
	}

	return id, nil
}

func getTokenFromHeader(authHeader string) (uuid.UUID, error) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		return uuid.UUID{}, errors.New("token not provided")
	}

	token, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid token format")
	}

	return token, nil
}

func isAuthorized(token, dbToken uuid.UUID) bool {
	return token == dbToken
}
