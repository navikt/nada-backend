package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/openapi"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Server struct {
	repo         *database.Repo
	log          *logrus.Entry
	oauth2Config oauth2.Config
}

func New(repo *database.Repo, oauth2Config oauth2.Config, log *logrus.Entry) *Server {
	return &Server{
		repo:         repo,
		log:          log,
		oauth2Config: oauth2Config,
	}
}

// GetDataproducts (GET /dataproducts)
func (s *Server) GetDataproducts(w http.ResponseWriter, r *http.Request) {
	dataproducts, err := s.repo.GetDataproducts(r.Context())
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

	dataproduct, err := s.repo.CreateDataproduct(r.Context(), newDataproduct)
	if err != nil {
		s.log.WithError(err).Error("Creating dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dataproduct); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct as JSON")
	}
}

// DeleteDataproduct (DELETE /dataproducts/{id})
func (s *Server) DeleteDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.repo.DeleteDataproduct(r.Context(), id); err != nil {
		s.log.WithError(err).Error("Deleting dataproduct")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateDataproduct (PUT /dataproducts/{id})
func (s *Server) UpdateDataproduct(w http.ResponseWriter, r *http.Request, id string) {
	var newDataproduct openapi.NewDataproduct
	if err := json.NewDecoder(r.Body).Decode(&newDataproduct); err != nil {
		s.log.WithError(err).Info("Decoding newDataproduct")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	dataproduct, err := s.repo.UpdateDataproduct(r.Context(), id, newDataproduct)
	if err != nil {
		s.log.WithError(err).Error("Updating dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dataproduct); err != nil {
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

	dataset, err := s.repo.CreateDataset(r.Context(), newDataset)
	if err != nil {
		s.log.WithError(err).Error("Creating dataset")
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

// DeleteDataset (DELETE /datasets/{id})
func (s *Server) DeleteDataset(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.repo.DeleteDataset(r.Context(), id); err != nil {
		s.log.WithError(err).Error("Deleting dataset")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

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

	dataset, err := s.repo.UpdateDataset(r.Context(), id, newDataset)
	if err != nil {
		s.log.WithError(err).Error("Updating dataset")
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

// (GET /search)
func (s *Server) Search(w http.ResponseWriter, r *http.Request, params openapi.SearchParams) {
	results, err := s.repo.Search(r.Context(), params.Q)
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

// (GET /userinfo)
func (s *Server) GetUserInfo(w http.ResponseWriter, r *http.Request) {

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

	var domain string
	if strings.Contains(r.URL.Host, "dev.intern.nav.no") {
		domain = "dev.intern.nav.no"
	} else if strings.Contains(r.URL.Host, "intern.nav.no") {
		domain = "intern.nav.no"
	}

	w.Header().Set("Set-Cookie", fmt.Sprintf("jwt=%v;HttpOnly;Secure;Max-Age=86400;Path=/;Domain=%v", tokens.AccessToken, domain))

	var loginPage string
	if strings.HasPrefix(r.URL.Host, "localhost") {
		loginPage = "http://localhost:3000/"
	} else {
		loginPage = fmt.Sprintf("https://%v", r.URL.Host)
	}

	http.Redirect(w, r, loginPage, http.StatusFound)
}
