package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/auth"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type GCP interface {
	GetDataset(ctx context.Context, projectID, datasetID string) ([]openapi.BigqueryTypeMetadata, error)
	GetDatasets(ctx context.Context, projectID string) ([]string, error)
	GetTables(ctx context.Context, projectID string) ([]gensql.DatasourceBigquery, error)
}

type OAuth2 interface {
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

type Server struct {
	repo            *database.Repo
	log             *logrus.Entry
	oauth2Config    OAuth2
	projectsMapping *auth.TeamProjectsUpdater
	gcp             GCP
	datasetEnricher DatasetEnricher
}

func New(
	repo *database.Repo,
	oauth2Config OAuth2,
	log *logrus.Entry,
	projectsMapping *auth.TeamProjectsUpdater,
	gcp GCP,
	enricher DatasetEnricher,
) *Server {
	return &Server{
		repo:            repo,
		log:             log,
		oauth2Config:    oauth2Config,
		projectsMapping: projectsMapping,
		gcp:             gcp,
		datasetEnricher: enricher,
	}
}

// GetCollections (GET /collections)
func (s *Server) GetCollections(w http.ResponseWriter, r *http.Request, params openapi.GetCollectionsParams) {
	dataproducts, err := s.repo.GetCollections(r.Context(), defaultInt(params.Limit, 15), defaultInt(params.Offset, 0))
	if err != nil {
		s.log.WithError(err).Error("Getting collections")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(dataproducts); err != nil {
		s.log.WithError(err).Error("Encoding collections as JSON")
	}
}

// GetCollection (GET /collections/{id})
func (s *Server) GetCollection(w http.ResponseWriter, r *http.Request, id string) {
	collection, err := s.repo.GetCollection(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "No collection", http.StatusNotFound)
			return
		}

		s.log.WithError(err).Error("Getting collection")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		s.log.WithError(err).Error("Encoding collection as JSON")
	}
}

// CreateCollection (POST /collections)
func (s *Server) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var newCollection openapi.NewCollection
	if err := json.NewDecoder(r.Body).Decode(&newCollection); err != nil {
		s.log.WithError(err).Info("Decoding new collection")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}
	user := auth.GetUser(r.Context())

	if !user.Groups.Contains(newCollection.Owner.Group) {
		s.log.Infof("Creating collection: User %v is not member of Group %v", user.Email, newCollection.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	collection, err := s.repo.CreateCollection(r.Context(), newCollection)
	if err != nil {
		s.log.WithError(err).Error("Creating collection")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Created collection: %v", collection.Name)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		s.log.WithError(err).Error("Encoding collection as JSON")
	}
}

// DeleteCollection (DELETE /collections/{id})
func (s *Server) DeleteCollection(w http.ResponseWriter, r *http.Request, id string) {
	user := auth.GetUser(r.Context())

	collection, err := s.repo.GetCollection(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Getting collection for deletion")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if !user.Groups.Contains(collection.Owner.Group) {
		s.log.Infof("Delete collection: User %v is not member of Group %v", user.Email, collection.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.repo.DeleteCollection(r.Context(), id); err != nil {
		s.log.WithError(err).Error("Deleting collection")
		return
	}

	s.log.Infof("Deleted collection: %v", collection.Name)
	w.WriteHeader(http.StatusNoContent)
}

// UpdateCollection (PUT /collections/{id})
func (s *Server) UpdateCollection(w http.ResponseWriter, r *http.Request, id string) {
	var in openapi.UpdateCollection
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		s.log.WithError(err).Info("Decoding updated collection")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	existing, err := s.repo.GetCollection(context.Background(), id)
	if err != nil {
		s.log.WithError(err).Info("Update collection")
		http.Error(w, "uh oh", http.StatusBadRequest)
		return
	}

	user := auth.GetUser(r.Context())
	if !user.Groups.Contains(existing.Owner.Group) {
		s.log.Infof("Update collection: User %v is not member of Group %v", user.Email, existing.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	updated, err := s.repo.UpdateCollection(r.Context(), id, in)
	if err != nil {
		s.log.WithError(err).Error("Updating collection")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Updated collection: %v", updated.Name)

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		s.log.WithError(err).Error("Encoding collection as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// CreateDataproduct (POST /dataproducts)
func (s *Server) CreateDataproduct(w http.ResponseWriter, r *http.Request) {
	var input openapi.NewDataproduct
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.log.WithError(err).Info("Decoding newDataset")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	user := auth.GetUser(r.Context())

	datasource, err := database.MapDatasource(input.Datasource)
	if err != nil {
		s.log.WithError(err).Info("Decoding datasource")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	if !s.projectsMapping.OwnsProject(input.Owner.Group, datasource.ProjectId) {
		s.log.Infof("Creating dataproduct: BigQuery project %v is not owned by Group %v", datasource.ProjectId, input.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !user.Groups.Contains(input.Owner.Group) {
		s.log.Infof("Creating dataproduct: User %v is not member of Group %v", user.Email, input.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	created, err := s.repo.CreateDataproduct(r.Context(), input)
	if err != nil {
		s.log.WithError(err).Error("Creating dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if bq, ok := created.Datasource.(openapi.Bigquery); ok {
		err := s.datasetEnricher.UpdateSchema(r.Context(), gensql.DatasourceBigquery{
			DataproductID: uuid.MustParse(created.Id),
			ProjectID:     bq.ProjectId,
			Dataset:       bq.Dataset,
			TableName:     bq.Table,
		})
		if err != nil {
			s.log.WithError(err).WithField("dataproduct", created.Id).Error("unable to update bigquery dataset schema")
		}
	}
	s.log.Infof("Created dataproduct: %v", created.Name)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// DeleteDataproduct (DELETE /dataproducts/{id})
func (s *Server) DeleteDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	user := auth.GetUser(r.Context())

	dp, err := s.repo.GetDataproduct(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Get dataproduct for checking permissions on delete dp")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if !user.Groups.Contains(dp.Owner.Group) {
		s.log.Infof("Deleting dataproduct: User %v is not member of Group %v", user.Email, dp.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.repo.DeleteDataproduct(r.Context(), id); err != nil {
		s.log.WithError(err).Error("Deleting dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Deleted dataproduct: %v", dp.Name)

	w.WriteHeader(http.StatusNoContent)
}

// GetDataproducts (GET /dataproductss/)
func (s *Server) GetDataproducts(w http.ResponseWriter, r *http.Request, params openapi.GetDataproductsParams) {
	dp, err := s.repo.GetDataproducts(r.Context(), defaultInt(params.Limit, 15), defaultInt(params.Offset, 0))
	if err != nil {
		s.log.WithError(err).Error("Get dataproducts")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dp); err != nil {
		s.log.WithError(err).Error("Encoding dataproducts as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// GetDataproduct (GET /dataproducts/{id})
func (s *Server) GetDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	dp, err := s.repo.GetDataproduct(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Get dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dp); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// UpdateDataproduct (PUT /dataproducts/{id})
func (s *Server) UpdateDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	var input openapi.UpdateDataproduct
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.log.WithError(err).Info("Decoding dataproduct")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	user := auth.GetUser(r.Context())

	existing, err := s.repo.GetDataproduct(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Getting dataproduct for checking permissions on update updated")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	datasource := existing.Datasource.(openapi.Bigquery)
	if !s.projectsMapping.OwnsProject(existing.Owner.Group, datasource.ProjectId) {
		s.log.Infof("Creating dataproduct: BigQuery project %v is not owned by Group %v", datasource.ProjectId, existing.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !user.Groups.Contains(existing.Owner.Group) {
		s.log.Infof("Updating dataproduct: User %v is not member of Group %v", user.Email, existing.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	updated, err := s.repo.UpdateDataproduct(r.Context(), id, input)
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

func (s *Server) GetDataproductMetadata(w http.ResponseWriter, r *http.Request, id string) {
	metadata, err := s.repo.GetDataproductMetadata(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Getting dataproduct metadata")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct metadata as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

func (s *Server) GetBigqueryDatasets(w http.ResponseWriter, r *http.Request, id string) {
	s.log.Info("hello world")
	ret, err := s.gcp.GetDatasets(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Getting BigQuery datasets")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ret); err != nil {
		s.log.WithError(err).Error("Encoding bigquery datasets as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

func (s *Server) GetBigqueryDataset(w http.ResponseWriter, r *http.Request, projectID, datasetID string) {
	ret, err := s.gcp.GetDataset(r.Context(), projectID, datasetID)
	if err != nil {
		s.log.WithError(err).Error("Getting BigQuery dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ret); err != nil {
		s.log.WithError(err).Error("Encoding bigquery dataset as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

func (s *Server) GetBigqueryTables(w http.ResponseWriter, r *http.Request, id string) {
	ret, err := s.gcp.GetTables(r.Context(), id)
	if err != nil {
		s.log.WithError(err).Error("Getting BigQuery tables")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ret); err != nil {
		s.log.WithError(err).Error("Encoding bigquery tables as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

func (s *Server) AddToCollection(w http.ResponseWriter, r *http.Request, collectionID string) {
	var body openapi.CollectionElement
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.log.WithError(err).Info("Decoding request body")
		http.Error(w, "uh oh", http.StatusBadRequest)
		return
	}

	user := auth.GetUser(r.Context())

	existing, err := s.repo.GetCollection(r.Context(), collectionID)
	if err != nil {
		s.log.WithError(err).Info("Getting collection for checking permissions on add to collection")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	if !user.Groups.Contains(existing.Owner.Group) {
		s.log.Infof("Add to collection: User %v is not member of Group %v", user.Email, existing.Owner.Group)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.repo.AddToCollection(r.Context(), collectionID, body); err != nil {
		s.log.WithError(err).Error("Adding to collection")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		s.log.WithError(err).Error("Encoding collection content as JSON")
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
	}

	for _, g := range user.Groups {
		userInfo.Groups = append(userInfo.Groups, openapi.Group{
			Email: g.Email,
			Name:  g.Name,
		})
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userInfo); err != nil {
		s.log.WithError(err).Error("Encoding userinfo as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	consentUrl := s.oauth2Config.AuthCodeURL("banan")
	http.Redirect(w, r, consentUrl, http.StatusFound)
}

func (s *Server) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Error(w, "No code in query params", http.StatusForbidden)
		return
	}

	// TODO(thokra): Introduce varying state
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

	rawIDToken, ok := tokens.Extra("id_token").(string)
	if !ok {
		s.log.Info("Missing id_token")
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	// Parse and verify ID Token payload.
	_, err = s.oauth2Config.Verify(r.Context(), rawIDToken)
	if err != nil {
		s.log.Info("Invalid id_token")
		http.Error(w, "uh oh", http.StatusForbidden)
		return
	}

	// TODO(thokra): Use secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    tokens.AccessToken + "|" + rawIDToken,
		Path:     "/",
		Domain:   r.Host,
		Expires:  tokens.Expiry,
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
