package api

import (
	"github.com/labstack/echo/v4"
	"github.com/navikt/datakatalogen/backend/database"
	"github.com/navikt/datakatalogen/backend/openapi"
)

type Server struct {
	repo database.Repo
}

func New(repo database.Repo) *Server {
	return &Server{
		repo: repo,
	}
}

// (GET /dataproducts)
func (s *Server) GetDataproducts(ctx echo.Context) error {
	return nil
}

// (POST /dataproducts)
func (s *Server) CreateDataproduct(ctx echo.Context) error {
	return nil
}

// (DELETE /dataproducts/{dataproduct_id})
func (s *Server) DeleteDataproduct(ctx echo.Context, dataproductId string) error {
	return nil
}

// (GET /dataproducts/{dataproduct_id})
func (s *Server) GetDataproduct(ctx echo.Context, dataproductId string) error {
	return nil
}

// (PUT /dataproducts/{dataproduct_id})
func (s *Server) UpdateDataproduct(ctx echo.Context, dataproductId string) error {
	return nil
}

// (GET /dataproducts/{dataproduct_id}/datasets)
func (s *Server) GetDatasets(ctx echo.Context, dataproductId string) error {
	return nil
}

// (POST /dataproducts/{dataproduct_id}/datasets)
func (s *Server) CreateDataset(ctx echo.Context, dataproductId string) error {
	return nil
}

// (DELETE /dataproducts/{dataproduct_id}/datasets/{dataset_id})
func (s *Server) DeleteDataset(ctx echo.Context, dataproductId string, datasetId string) error {
	return nil
}

// (GET /dataproducts/{dataproduct_id}/datasets/{dataset_id})
func (s *Server) GetDataset(ctx echo.Context, dataproductId string, datasetId string) error {
	return nil
}

// (PUT /dataproducts/{dataproduct_id}/datasets/{dataset_id})
func (s *Server) UpdateDataset(ctx echo.Context, dataproductId string, datasetId string) error {
	return nil
}

// (GET /search)
func (s *Server) Search(ctx echo.Context, params openapi.SearchParams) error {
	return nil
}
