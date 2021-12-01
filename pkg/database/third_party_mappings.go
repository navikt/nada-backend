package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetDataproductMappings(ctx context.Context, dataproductID uuid.UUID) ([]models.MappingService, error) {
	tpm, err := r.querier.GetDataproductMappings(ctx, dataproductID)
	if err != nil {
		return nil, err
	}

	svcs := []models.MappingService{}
	for _, s := range tpm.Services {
		svcs = append(svcs, models.MappingService(s))
	}

	return svcs, nil
}

func (r *Repo) GetDataproductsByMapping(ctx context.Context, service models.MappingService) ([]*models.Dataproduct, error) {
	dataproducts := []*models.Dataproduct{}
	dps, err := r.querier.GetDataproductsByMapping(ctx, string(service))
	if err != nil {
		return nil, err
	}

	for _, entry := range dps {
		dataproducts = append(dataproducts, dataproductFromSQL(entry))
	}

	return dataproducts, nil
}

func (r *Repo) MapDataproduct(ctx context.Context, dataproductID uuid.UUID, services []models.MappingService) error {
	svcs := []string{}
	for _, s := range services {
		svcs = append(svcs, string(s))
	}
	return r.querier.MapDataproduct(ctx, gensql.MapDataproductParams{
		DataproductID: dataproductID,
		Services:      svcs,
	})
}
