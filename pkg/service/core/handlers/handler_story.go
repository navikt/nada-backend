package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/amplitude"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/service"
	log "github.com/sirupsen/logrus"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	idURLPosUpdate         = 3
	idURLPosGet            = 2
	maxMemoryMultipartForm = 32 << 20 // 32 MB
)

type storyHandler struct {
	storyService    service.StoryService
	tokenService    service.TokenService
	amplitudeClient amplitude.Amplitude
}

func (h *storyHandler) GetGCSObject(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	objPath := strings.Join(pathParts[2:], "/")

	attr, objBytes, err := h.storyService.GetObject(r.Context(), objPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if strings.HasSuffix(objPath, ".js") {
		w.Header().Add("content-type", "text/javascript")
	} else if strings.HasSuffix(objPath, ".css") {
		w.Header().Add("content-type", "text/css")
	} else {
		w.Header().Add("content-type", attr.ContentType)
	}

	w.Header().Add("content-length", strconv.Itoa(int(attr.Size)))
	w.Header().Add("content-encoding", attr.ContentEncoding)

	// FIXME: is this correct?
	_, _ = w.Write(objBytes)
}

func (h *storyHandler) CreateStoryHTTP(w http.ResponseWriter, r *http.Request) {
	team := r.Context().Value("team").(string)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Errorf("reading body")
		writeError(w, http.StatusBadRequest, fmt.Errorf("error reading body"))
		return
	}

	newStory := service.NewStory{}
	if err := json.Unmarshal(bodyBytes, &newStory); err != nil {
		log.WithError(err).Errorf("unmarshalling request body")
		writeError(w, http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"))
		return
	}

	newStory.Group = team
	if newStory.Keywords == nil {
		newStory.Keywords = []string{}
	}

	story, err := h.storyService.CreateStoryWithTeamAndProductArea(r.Context(), &newStory)
	if err != nil {
		log.WithError(err).Errorf("creating story")
		writeError(w, http.StatusInternalServerError, fmt.Errorf("error creating story"))
		return
	}

	retBytes, err := json.Marshal(story)
	if err != nil {
		log.WithError(err).Errorf("marshalling response json after creating story")
		writeError(w, http.StatusInternalServerError, fmt.Errorf("error creating story"))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(retBytes)
}

func (h *storyHandler) UpdateStoryHTTP(w http.ResponseWriter, r *http.Request) {
	qID, err := getIDFromPath(r, idURLPosUpdate)
	if err != nil {
		log.WithError(err).Errorf("getting story id from url path")
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid story id %v", qID))
		return
	}

	files, err := filesFromRequest(r)
	if err != nil {
		log.WithError(err).Errorf("reading files from request")
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request form"))
		return
	}

	err = h.storyService.RecreateStoryFiles(r.Context(), qID.String(), files)
	if err != nil {
		log.WithError(err).Errorf("uploading file")
		writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
	}
}

func (h *storyHandler) AppendStoryHTTP(w http.ResponseWriter, r *http.Request) {
	qID, err := getIDFromPath(r, idURLPosUpdate)
	if err != nil {
		log.WithError(err).Errorf("getting story id from url path")
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid story id %v", qID))
		return
	}
}

// FIXME: take a closer look at this, maybe we can do it a bit differently
func StoryHTTPMiddleware(h *storyHandler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				h.getStoryHTTP(w, r, next)
			case http.MethodPost:
				h.createStoryHTTP(w, r, next)
			case http.MethodPut:
				fallthrough
			case http.MethodPatch:
				h.updateStoryHTTP(w, r, next)
			}
		})
	}
}

func (h *storyHandler) updateStoryHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	qID, err := getIDFromPath(r, idURLPosUpdate)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid story id %v", qID))
		return
	}

	authHeader := r.Header.Get("Authorization")
	token, err := getTokenFromHeader(authHeader)
	if err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}

	story, apiErr := h.storyService.GetStory(r.Context(), qID)
	if apiErr != nil {
		if errors.Is(apiErr, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, fmt.Errorf("story id %v does not exist", qID))
			return
		}

		log.WithError(apiErr).Errorf("reading story id %v", qID)
		writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	group := strings.Split(story.Group, "@")[0]
	dbToken, err := h.tokenService.GetTeamFromNadaToken(r.Context(), auth.TrimNaisTeamPrefix(group))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Errorf("no nada token found for team %v, story id %v", story.Group, qID)
			writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			return
		}

		log.WithError(err).Errorf("reading nada token for group %v, story id %v", story.Group, qID)
		writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	if !isAuthorized(token.String(), dbToken) {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized to update story %v", qID))
		return
	}

	next.ServeHTTP(w, r)
}

func isAuthorized(token, dbToken string) bool {
	return token == dbToken
}

func (h *storyHandler) createStoryHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	authHeader := r.Header.Get("Authorization")
	token, err := getTokenFromHeader(authHeader)
	if err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}

	team, err := h.tokenService.GetTeamFromNadaToken(r.Context(), token.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusForbidden, errors.New("no nada teams correspond to the team token provided with the request"))
		} else {
			writeError(w, http.StatusForbidden, err)
		}
		return
	}

	ctx := context.WithValue(r.Context(), "team", team+"@nav.no")
	next.ServeHTTP(w, r.WithContext(ctx))
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

func (h *storyHandler) RedirectStoryHTTP(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	urlPathPrefix := strings.Join(pathParts[0:2], "/") + "/"
	storyPath := strings.Join(pathParts[2:], "/")

	objPath, err := h.storyService.GetIndexHtmlPath(r.Context(), storyPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.URL.Path = urlPathPrefix
	http.Redirect(w, r, objPath, http.StatusSeeOther)
}

func (h *storyHandler) getStoryHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	regex, _ := regexp.Compile(`[\n]*\.[\n]*`) // check if object path has file extension
	if !regex.MatchString(r.URL.Path) {
		h.RedirectStoryHTTP(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, ".html") {
		if err := h.publishAmplitudeEvent(r.Context(), r.URL.Path); err != nil {
			log.WithError(err).Info("Failed to publish amplitude event")
		}
	}

	next.ServeHTTP(w, r)
}

// FIXME: this mustParse stuff isn't great
func (h *storyHandler) publishAmplitudeEvent(ctx context.Context, path string) error {
	id := strings.Split(path, "/")[2]

	story, err := h.storyService.GetStory(ctx, uuid.MustParse(id))
	if err != nil {
		return err
	}

	if err := h.amplitudeClient.PublishEvent(ctx, story.Name); err != nil {
		return err
	}

	return nil
}

func filesFromRequest(r *http.Request) ([]*service.UploadFile, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("creating multipart reader: %w", err)
	}

	var files []*service.UploadFile
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading next part: %w", err)
		}

		if part.FileName() == "" {
			continue
		}

		data, err := io.ReadAll(part)
		if err != nil {
			return nil, fmt.Errorf("reading part data: %w", err)
		}

		fileFullPath := part.FileName()

		// try to extract full path from content-disposition header
		_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err == nil {
			pathInCDHeader := params["name"]
			if pathInCDHeader != "" {
				fileFullPath = pathInCDHeader
			}
		}

		files = append(files, &service.UploadFile{
			Path: fileFullPath,
			Data: data,
		})
	}

	return files, nil
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

func writeError(w http.ResponseWriter, status int, err error) {
	resp := map[string]string{
		"statusCode": strconv.Itoa(status),
		"message":    err.Error(),
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.WithError(err).Errorf("marshalling error response")
	}

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func NewStoryHandler(storyService service.StoryService, tokenService service.TokenService, amp amplitude.Amplitude) *storyHandler {
	return &storyHandler{
		storyService:    storyService,
		tokenService:    tokenService,
		amplitudeClient: amp,
	}
}
