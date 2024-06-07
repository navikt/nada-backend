package service

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
	"github.com/navikt/nada-backend/pkg/auth"
)

const (
	idURLPosUpdate         = 3
	idURLPosGet            = 2
	maxMemoryMultipartForm = 32 << 20 // 32 MB
)

func GetGCSObject(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	objPath := strings.Join(pathParts[2:], "/")

	attr, objBytes, err := gcsClient.GetObject(r.Context(), objPath)
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

	w.Write(objBytes)
}

func CreateStoryHTTP(w http.ResponseWriter, r *http.Request) {
	team := r.Context().Value("team").(string)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Errorf("reading body")
		writeError(w, http.StatusBadRequest, fmt.Errorf("error reading body"))
		return
	}

	newStory := NewStory{}
	if err := json.Unmarshal(bodyBytes, &newStory); err != nil {
		log.WithError(err).Errorf("unmarshalling request body")
		writeError(w, http.StatusBadRequest, fmt.Errorf("error unmarshalling request body"))
		return
	}
	newStory.Group = team
	if newStory.Keywords == nil {
		newStory.Keywords = []string{}
	}

	if err := setProductAreaAndTeamCatalogURL(r.Context(), &newStory); err != nil {
		log.WithError(err).Errorf("setting product area and team catalog URL")
		writeError(w, http.StatusBadRequest, fmt.Errorf("error creating story, unable to get team from teamcatalogue corresponding to team id '%v'", *newStory.TeamID))
		return
	}

	story, err := dbCreateStory(r.Context(), team, &newStory)
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

func UpdateStoryHTTP(w http.ResponseWriter, r *http.Request) {
	qID, err := getIDFromPath(r, idURLPosUpdate)
	if err != nil {
		log.WithError(err).Errorf("getting story id from url path")
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid story id %v", qID))
		return
	}

	err = r.ParseMultipartForm(maxMemoryMultipartForm)
	if err != nil {
		log.WithError(err).Errorf("parsing multipart form")
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request form"))
		return
	}

	// Delete the root directory before uploading new files
	if err = gcsClient.DeleteObjectsWithPrefix(r.Context(), qID.String()); err != nil {
		log.WithError(err).Errorf("deleting objects with prefix")
		writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	for _, fileHeader := range r.MultipartForm.File {
		if err := uploadFileHTTP(r.Context(), qID.String(), fileHeader); err != nil {
			log.WithError(err).Errorf("uploading file")
			writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			return
		}
	}
}

func AppendFileHTTP(w http.ResponseWriter, r *http.Request) {
	qID, err := getIDFromPath(r, idURLPosUpdate)
	if err != nil {
		log.WithError(err).Errorf("getting story id from url path")
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid story id %v", qID))
		return
	}

	err = r.ParseMultipartForm(maxMemoryMultipartForm)
	if err != nil {
		log.WithError(err).Errorf("parsing multipart form")
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request form"))
		return
	}

	for _, fileHeader := range r.MultipartForm.File {
		if err := uploadFileHTTP(r.Context(), qID.String(), fileHeader); err != nil {
			log.WithError(err).Errorf("uploading file")
			writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			return
		}
	}
}

func RedirectStoryHTTP(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	urlPathPrefix := strings.Join(pathParts[0:2], "/") + "/"
	storyPath := strings.Join(pathParts[2:], "/")

	objPath, err := gcsClient.GetIndexHtmlPath(r.Context(), storyPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.URL.Path = urlPathPrefix
	http.Redirect(w, r, objPath, http.StatusSeeOther)
}

func StoryHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getStoryHTTP(w, r, next)
		case http.MethodPost:
			createStoryHTTP(w, r, next)
		case http.MethodPut:
			fallthrough
		case http.MethodPatch:
			updateStoryHTTP(w, r, next)
		}
	})
}

func getStoryHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	regex, _ := regexp.Compile(`[\n]*\.[\n]*`) // check if object path has file extension
	if !regex.MatchString(r.URL.Path) {
		RedirectStoryHTTP(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, ".html") {
		if err := publishAmplitudeEvent(r.Context(), r.URL.Path); err != nil {
			log.WithError(err).Info("Failed to publish amplitude event")
		}
	}

	next.ServeHTTP(w, r)
}

func createStoryHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	authHeader := r.Header.Get("Authorization")
	token, err := getTokenFromHeader(authHeader)
	if err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}

	team, err := GetTeamFromToken(r.Context(), token)
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

func updateStoryHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
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

	story, apiErr := GetStoryMetadata(r.Context(), qID.String())
	if apiErr != nil {
		if errors.Is(apiErr.Err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, fmt.Errorf("story id %v does not exist", qID))
			return
		}

		log.WithError(apiErr.Err).Errorf("reading story id %v", qID)
		writeError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	group := strings.Split(story.Group, "@")[0]
	dbToken, err := GetNadaToken(r.Context(), auth.TrimNaisTeamPrefix(group))
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

	if !isAuthorized(token, dbToken) {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized to update story %v", qID))
		return
	}

	next.ServeHTTP(w, r)
}

func uploadFileHTTP(ctx context.Context, objPath string, fileHeader []*multipart.FileHeader) error {
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

		if err := gcsClient.UploadFile(ctx, objPath+"/"+fileFullPath, file); err != nil {
			return err
		}
	}
	return nil
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

func publishAmplitudeEvent(ctx context.Context, urlPath string) error {
	id := strings.Split(urlPath, "/")[2]
	story, err := GetStoryMetadata(ctx, id)
	if err != nil {
		return err
	}
	if err := amplitudeClient.PublishEvent(ctx, story.Name); err != nil {
		return err
	}
	return nil
}

func setProductAreaAndTeamCatalogURL(ctx context.Context, newStory *NewStory) error {
	if newStory.TeamID == nil {
		log.Warningf("team id not provided for story %v", newStory.Name)
		return nil
	}

	teamCatalogURL := tkClient.GetTeamCatalogURL(*newStory.TeamID)
	team, err := tkClient.GetTeam(ctx, *newStory.TeamID)
	if err != nil {
		return err
	}

	newStory.TeamkatalogenURL = &teamCatalogURL
	newStory.ProductAreaID = &team.ProductAreaID

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
