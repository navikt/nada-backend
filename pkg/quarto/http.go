package quarto

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/gcs"
	"github.com/navikt/nada-backend/pkg/graph"
	"github.com/navikt/nada-backend/pkg/graph/models"
	"github.com/sirupsen/logrus"
)

const (
	idURLPosUpdate         = 3
	idURLPosGet            = 2
	maxMemoryMultipartForm = 32 << 20 // 32 MB
)

type Handler struct {
	repo            *database.Repo
	gcsClient       *gcs.Client
	teamCatalog     graph.Teamkatalogen
	amplitudeClient amplitude.Amplitude
	log             *logrus.Entry
}

func NewHandler(repo *database.Repo, gcsClient *gcs.Client, teamCatalog graph.Teamkatalogen, amplitudeClient amplitude.Amplitude, logger *logrus.Entry) *Handler {
	return &Handler{
		repo:            repo,
		gcsClient:       gcsClient,
		teamCatalog:     teamCatalog,
		amplitudeClient: amplitudeClient,
		log:             logger,
	}
}

func (h *Handler) GetObject(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/quarto/")

	attr, objBytes, err := h.gcsClient.GetObject(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if strings.HasSuffix(path, ".js") {
		w.Header().Add("content-type", "text/javascript")
	} else if strings.HasSuffix(path, ".css") {
		w.Header().Add("content-type", "text/css")
	} else {
		w.Header().Add("content-type", attr.ContentType)
	}

	w.Header().Add("content-length", strconv.Itoa(int(attr.Size)))
	w.Header().Add("content-encoding", attr.ContentEncoding)

	w.Write(objBytes)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	team := r.Context().Value("team").(string)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.WithError(err).Errorf("reading body")
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("error reading body"))
		return
	}

	newQuartoStory := models.NewQuartoStory{}
	if err := json.Unmarshal(bodyBytes, &newQuartoStory); err != nil {
		h.log.WithError(err).Errorf("unmarshalling request body")
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"))
		return
	}
	newQuartoStory.Group = team
	if newQuartoStory.Keywords == nil {
		newQuartoStory.Keywords = []string{}
	}

	if err := h.setProductAreaAndTeamCatalogURL(r.Context(), &newQuartoStory); err != nil {
		h.log.WithError(err).Errorf("setting product area and team catalog URL")
	}

	quarto, err := h.repo.CreateQuartoStory(r.Context(), team, newQuartoStory)
	if err != nil {
		h.log.WithError(err).Errorf("creating quarto story")
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("error creating quarto story"))
		return
	}

	retBytes, err := json.Marshal(quarto)
	if err != nil {
		h.log.WithError(err).Errorf("marshalling response json after creating quarto")
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("error creating quarto story"))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(retBytes)
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

	// Delete the root directory before uploading new files
	if err = h.gcsClient.DeleteObjectsWithPrefix(r.Context(), qID.String()); err != nil {
		h.log.WithError(err).Errorf("deleting objects with prefix")
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
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

func (h *Handler) Append(w http.ResponseWriter, r *http.Request) {
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
	objPath, err := h.gcsClient.GetIndexHtmlPath(r.Context(), strings.TrimPrefix(r.URL.Path, "/quarto/"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.URL.Path = "/quarto/"
	http.Redirect(w, r, objPath, http.StatusSeeOther)
}

func (h *Handler) QuartoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.getQuarto(w, r, next)
		case http.MethodPost:
			h.createQuarto(w, r, next)
		case http.MethodPut:
			fallthrough
		case http.MethodPatch:
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

	if strings.HasSuffix(r.URL.Path, ".html") {
		if err := h.publishAmplitudeEvent(r.Context(), r.URL.Path); err != nil {
			h.log.WithError(err).Warning("Failed to publish amplitude event")
		}
	}

	next.ServeHTTP(w, r)
}

func (h *Handler) createQuarto(w http.ResponseWriter, r *http.Request, next http.Handler) {
	authHeader := r.Header.Get("Authorization")
	token, err := getTokenFromHeader(authHeader)
	if err != nil {
		h.writeError(w, http.StatusForbidden, err)
		return
	}

	team, err := h.repo.GetTeamFromToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusForbidden, err)
		return
	}

	ctx := context.WithValue(r.Context(), "team", team+"@nav.no")
	next.ServeHTTP(w, r.WithContext(ctx))
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
		fileFullPath := f.Filename

		// try to extract full path from content-disposition header
		_, params, err := mime.ParseMediaType(f.Header.Get("Content-Disposition"))
		if err == nil {
			pathInCDHeader := params["name"]
			if pathInCDHeader != "" {
				fileFullPath = pathInCDHeader
			}
		}

		file, err := f.Open()
		if err != nil {
			return err
		}

		h.log.Printf("upload quarto file full path %v", objPath+"/"+fileFullPath)

		if err := h.gcsClient.UploadFile(ctx, objPath+"/"+fileFullPath, file); err != nil {
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

func (h *Handler) publishAmplitudeEvent(ctx context.Context, urlPath string) error {
	id := strings.Split(urlPath, "/")[2]
	story, err := h.repo.GetQuartoStory(ctx, uuid.MustParse(id))
	if err != nil {
		return err
	}
	if err := h.amplitudeClient.PublishEvent(ctx, story.Name); err != nil {
		return err
	}
	return nil
}

func (h *Handler) setProductAreaAndTeamCatalogURL(ctx context.Context, newQuartoStory *models.NewQuartoStory) error {
	if newQuartoStory.TeamID == nil {
		h.log.Warningf("team id not provided for quarto story %v", newQuartoStory.Name)
		return nil
	}

	teamCatalogURL := h.teamCatalog.GetTeamCatalogURL(*newQuartoStory.TeamID)
	team, err := h.teamCatalog.GetTeam(ctx, *newQuartoStory.TeamID)
	if err != nil {
		return err
	}

	newQuartoStory.TeamkatalogenURL = &teamCatalogURL
	newQuartoStory.ProductAreaID = &team.ProductAreaID

	return nil
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
