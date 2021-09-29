package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/navikt/datakatalogen/backend/database"
	"github.com/navikt/datakatalogen/backend/openapi"
)

type Server struct {
	repo *database.Repo
}

func New(repo *database.Repo) *Server {
	return &Server{
		repo: repo,
	}
}

// (GET /dataproducts)
func (s *Server) GetDataproducts(w http.ResponseWriter, r *http.Request) {
	res, err := s.repo.GetDataproducts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Println(err)
	}
}

// (POST /dataproducts)
func (s *Server) CreateDataproduct(w http.ResponseWriter, r *http.Request) {
}

// (DELETE /dataproducts/{dataproduct_id})
func (s *Server) DeleteDataproduct(w http.ResponseWriter, r *http.Request, dataproductId string) {
}

// (GET /dataproducts/{dataproduct_id})
func (s *Server) GetDataproduct(w http.ResponseWriter, r *http.Request, dataproductId string) {
}

// (PUT /dataproducts/{dataproduct_id})
func (s *Server) UpdateDataproduct(w http.ResponseWriter, r *http.Request, dataproductId string) {
}

// (GET /dataproducts/{dataproduct_id}/datasets)
func (s *Server) GetDatasets(w http.ResponseWriter, r *http.Request, dataproductId string) {
}

// (POST /dataproducts/{dataproduct_id}/datasets)
func (s *Server) CreateDataset(w http.ResponseWriter, r *http.Request, dataproductId string) {
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
