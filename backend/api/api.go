package api

import (
	"encoding/json"
	"net/http"

	"github.com/navikt/datakatalogen/backend/database"
	"github.com/navikt/datakatalogen/backend/openapi"
	"github.com/sirupsen/logrus"
)

type Server struct {
	repo *database.Repo
	log  *logrus.Entry
}

func New(repo *database.Repo, log *logrus.Entry) *Server {
	return &Server{
		repo: repo,
		log:  log,
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

// GetDataproduct (GET /dataproducts/{dataproduct_id})
func (s *Server) GetDataproduct(w http.ResponseWriter, r *http.Request, dataproductId string) {
	dataproduct, err := s.repo.GetDataproduct(r.Context(), dataproductId)
	if err != nil {
		s.log.WithError(err).Error("Getting dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(dataproduct); err != nil {
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
	if err = json.NewEncoder(w).Encode(dataproduct); err != nil {
		s.log.WithError(err).Error("Encoding dataproduct as JSON")
	}
}

// DeleteDataproduct (DELETE /dataproducts/{dataproduct_id})
func (s *Server) DeleteDataproduct(w http.ResponseWriter, r *http.Request, dataproductId string) {
	if err := s.repo.DeleteDataproduct(r.Context(), dataproductId); err != nil {
		s.log.WithError(err).Error("Deleting dataproduct")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateDataproduct (PUT /dataproducts/{dataproduct_id})
func (s *Server) UpdateDataproduct(w http.ResponseWriter, r *http.Request, dataproductId string) {
	var newDataproduct openapi.NewDataproduct
	if err := json.NewDecoder(r.Body).Decode(&newDataproduct); err != nil {
		s.log.WithError(err).Info("Decoding newDataproduct")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	dataproduct, err := s.repo.UpdateDataproduct(r.Context(), dataproductId, newDataproduct)
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

// GetDatasetsForDataproduct (GET /dataproducts/{dataproduct_id}/datasets)
func (s *Server) GetDatasetsForDataproduct(w http.ResponseWriter, r *http.Request, dataproductId string) {
	datasets, err := s.repo.GetDatasetsForDataproduct(r.Context(), dataproductId)
	if err != nil {
		s.log.WithError(err).Error("Getting datasets for dataproduct")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(datasets); err != nil {
		s.log.WithError(err).Error("Encoding datasets as JSON")
		http.Error(w, "uh oh", http.StatusInternalServerError)
		return
	}
}

// (POST /dataproducts/{dataproduct_id}/datasets)
func (s *Server) CreateDataset(w http.ResponseWriter, r *http.Request, dataproductId string) {
	var newDataset openapi.NewDataset
	if err := json.NewDecoder(r.Body).Decode(&newDataset); err != nil {
		s.log.WithError(err).Info("Decoding newDataset")
		http.Error(w, "invalid JSON object", http.StatusBadRequest)
		return
	}

	dataset, err := s.repo.CreateDataset(r.Context(), dataproductId, newDataset)
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

// (DELETE /dataproducts/{dataproduct_id}/datasets/{dataset_id})
func (s *Server) DeleteDataset(w http.ResponseWriter, r *http.Request, dataproductId string, datasetId string) {
}

// (GET /dataproducts/{dataproduct_id}/datasets/{dataset_id})
func (s *Server) GetDataset(w http.ResponseWriter, r *http.Request, dataproductId string, datasetId string) {
}

// (PUT /dataproducts/{dataproduct_id}/datasets/{dataset_id})
func (s *Server) UpdateDataset(w http.ResponseWriter, r *http.Request, dataproductId string, datasetId string) {
}

// (GET /search)
func (s *Server) Search(w http.ResponseWriter, r *http.Request, params openapi.SearchParams) {
}
