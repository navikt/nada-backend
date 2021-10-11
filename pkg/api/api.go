package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Server struct {
	repo            *database.Repo
	log             *logrus.Entry
	oauth2Config    oauth2.Config
	projectsMapping *auth.TeamProjectsUpdater
}

func New(repo *database.Repo, oauth2Config oauth2.Config, log *logrus.Entry, projectsMapping *auth.TeamProjectsUpdater) *Server {
	return &Server{
		repo:            repo,
		log:             log,
		oauth2Config:    oauth2Config,
		projectsMapping: projectsMapping,
	}
}

// GetDataproducts (GET /dataproducts)
func (s *Server) GetDataproducts(w http.ResponseWriter, r *http.Request, params openapi.GetDataproductsParams) {
	dataproducts, err := s.repo.GetDataproducts(r.Context(), defaultInt(params.Limit, 15), defaultInt(params.Offset, 0))
	if err != nil {
		s.log.WithError(err).Error("Getting dataproducts")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(dataproducts); err != nil {
		s.log.WithError(err).Error("Encoding dataproducts as JSON")
	}
}

// GetDataproduct (GET /dataproducts/{id})
func (s *Server) GetDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	dataproduct, err := s.repo.GetDataproduct(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "No dataproduct", http.StatusNotFound)
			return
		}

		s.log.WithError(err).Error("Getting dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dataproduct); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct as JSON")
	}
}

// CreateDataproduct (POST /dataproducts)
func (s *Server) CreateDataproduct(w http.ResponseWriter, r *http.Request) {
	var newDataproduct openapi.NewDataproduct
	if err := json.NewDecoder(r.Body).Decode(&newDataproduct); err != nil {
		s.log.WithError(err).Info("Decoding newDataproduct")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}
	user := auth.GetUser(r.Context())

	if !contains(newDataproduct.Owner.Team, user.Teams) {
		s.log.Infof("Creating dataproduct: User %v is not member of team %v", user.Email, newDataproduct.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	dataproduct, err := s.repo.CreateDataproduct(r.Context(), newDataproduct)
	if err != nil {
		s.log.WithError(err).Error("Creating dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Created dataproduct: %v", dataproduct.Name)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dataproduct); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct as JSON")
	}
}

// DeleteDataproduct (DELETE /dataproducts/{id})
func (s *Server) DeleteDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	user := auth.GetUser(r.Context())

	dataproduct, err := s.repo.GetDataproduct(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Getting dataproduct for deletion")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if !contains(dataproduct.Owner.Team, user.Teams) {
		s.log.Infof("Delete dataproduct: User %v is not member of team %v", user.Email, dataproduct.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.repo.DeleteDataproduct(r.Context(), id); err != nil {
		s.log.WithError(err).Error("Deleting dataproduct")
		return
	}

	s.log.Infof("Deleted dataproduct: %v", dataproduct.Name)
	w.WriteHeader(http.StatusNoContent)
}

// UpdateDataproduct (PUT /dataproducts/{id})
func (s *Server) UpdateDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	var in openapi.UpdateDataproduct
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		s.log.WithError(err).Info("Decoding updatedDataproduct")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	existing, err := s.repo.GetDataproduct(context.Background(), id)
	if err != nil {
		s.log.WithError(err).Info("Update dataproduct")
		http.Error(w, "uh oh", http.StatusBadRequest)
		return
	}

	user := auth.GetUser(r.Context())
	if !contains(existing.Owner.Team, user.Teams) {
		s.log.Infof("Update dataproduct: User %v is not member of team %v", user.Email, existing.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	updated, err := s.repo.UpdateDataproduct(r.Context(), id, in)
	if err != nil {
		s.log.WithError(err).Error("Updating dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Updated dataproduct: %v", updated.Name)

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// CreateDataset (POST /datasets)
func (s *Server) CreateDataset(w http.ResponseWriter, r *http.Request) {
	var newDataset openapi.NewDataset
	if err := json.NewDecoder(r.Body).Decode(&newDataset); err != nil {
		s.log.WithError(err).Info("Decoding newDataset")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	user := auth.GetUser(r.Context())

	dataproduct, err := s.repo.GetDataproduct(r.Context(), newDataset.DataproductId)
	if err != nil {
		s.log.WithError(err).Error("Getting dataproduct for checking permissions on create dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if !s.projectsMapping.OwnsProject(dataproduct.Owner.Team, newDataset.Bigquery.ProjectId) {
		s.log.Infof("Creating dataset: BigQuery project %v is not owned by team %v", newDataset.Bigquery.ProjectId, dataproduct.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !contains(dataproduct.Owner.Team, user.Teams) {
		s.log.Infof("Creating dataset: User %v is not member of team %v", user.Email, dataproduct.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	dataset, err := s.repo.CreateDataset(r.Context(), newDataset)
	if err != nil {
		s.log.WithError(err).Error("Creating dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Created dataset: %v", dataset.Name)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dataset); err != nil {
		s.log.WithError(err).Error("Encoding dataset as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// DeleteDataset (DELETE /datasets/{id})
func (s *Server) DeleteDataset(w http.ResponseWriter, r *http.Request, id string) {
	user := auth.GetUser(r.Context())

	dataset, err := s.repo.GetDataset(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Get dataset for checking permissions on delete dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	dataproduct, err := s.repo.GetDataproduct(r.Context(), dataset.DataproductId)
	if err != nil {
		s.log.WithError(err).Error("Getting dataproduct for checking permissions on delete dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if !contains(dataproduct.Owner.Team, user.Teams) {
		s.log.Infof("Deleting dataset: User %v is not member of team %v", user.Email, dataproduct.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.repo.DeleteDataset(r.Context(), id); err != nil {
		s.log.WithError(err).Error("Deleting dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Deleted dataset: %v", dataset.Name)

	w.WriteHeader(http.StatusNoContent)
}

// GetDataset (GET /datasets/{id})
func (s *Server) GetDataset(w http.ResponseWriter, r *http.Request, id string) {
	dataset, err := s.repo.GetDataset(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Get dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dataset); err != nil {
		s.log.WithError(err).Error("Encoding dataset as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// UpdateDataset (PUT /datasets/{id})
func (s *Server) UpdateDataset(w http.ResponseWriter, r *http.Request, id string) {
	var newDataset openapi.NewDataset
	if err := json.NewDecoder(r.Body).Decode(&newDataset); err != nil {
		s.log.WithError(err).Info("Decoding newDataset")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	user := auth.GetUser(r.Context())

	dataproduct, err := s.repo.GetDataproduct(r.Context(), newDataset.DataproductId)
	if err != nil {
		s.log.WithError(err).Error("Getting dataproduct for checking permissions on update dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if !s.projectsMapping.OwnsProject(dataproduct.Owner.Team, newDataset.Bigquery.ProjectId) {
		s.log.Infof("Creating dataset: BigQuery project %v is not owned by team %v", newDataset.Bigquery.ProjectId, dataproduct.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !contains(dataproduct.Owner.Team, user.Teams) {
		s.log.Infof("Updating dataset: User %v is not member of team %v", user.Email, dataproduct.Owner.Team)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	dataset, err := s.repo.UpdateDataset(r.Context(), id, newDataset)
	if err != nil {
		s.log.WithError(err).Error("Updating dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Updated dataset: %v", dataset.Name)

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dataset); err != nil {
		s.log.WithError(err).Error("Encoding dataset as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

func (s *Server) GetDatasetMetadata(w http.ResponseWriter, r *http.Request, id string) {
	metadata, err := s.repo.GetDatasetMetadata(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Getting dataset metadata")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		s.log.WithError(err).Error("Encoding datasetmetadata as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// Search (GET /search)
func (s *Server) Search(w http.ResponseWriter, r *http.Request, params openapi.SearchParams) {
	q := ""
	if params.Q != nil {
		q = *params.Q
	}
	results, err := s.repo.Search(r.Context(), q, defaultInt(params.Limit, 15), defaultInt(params.Offset, 0))
	if err != nil {
		s.log.WithError(err).Error("Search")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		s.log.WithError(err).Error("Encoding search result as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// UserInfo (GET /userinfo)
func (s *Server) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())

	userInfo := openapi.UserInfo{
		Email: user.Email,
		Name:  user.Name,
		Teams: user.Teams,
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userInfo); err != nil {
		s.log.WithError(err).Error("Encoding userinfo as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	consentUrl := s.oauth2Config.AuthCodeURL("banan", oauth2.SetAuthURLParam("redirect_uri", s.oauth2Config.RedirectURL))
	http.Redirect(w, r, consentUrl, http.StatusFound)
}

func (s *Server) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Error(w, "No code in query params", http.StatusForbidden)
		return
	}

	state := r.URL.Query().Get("state")
	if state != "banan" {
		s.log.Info("Incoming state does not match local state")
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	tokens, err := s.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		s.log.Errorf("Exchanging authorization code for tokens: %v", err)
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    tokens.AccessToken,
		Path:     "/",
		Domain:   r.Host,
		MaxAge:   86400,
		Secure:   true,
		HttpOnly: true,
	})

	var loginPage string
	if strings.HasPrefix(r.Host, "localhost") {
		loginPage = "http://localhost:3000/"
	} else {
		loginPage = "/"
	}
	http.Redirect(w, r, loginPage, http.StatusFound)
}

func defaultInt(i *int, def int) int {
	if i != nil {
		return *i
	}
	return def
}

func contains(elem string, list []string) bool {
	for _, entry := range list {
		if entry == elem {
			return true
		}
	}
	return false
}
