package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) GetDataproductMappings(ctx context.Context, dataproductID uuid.UUID) ([]models.MappingService, error) {
	tpm, err := r.querier.GetDataproductMappings(ctx, dataproductID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []models.MappingService{}, nil
		}
		return nil, err
	}

	svcs := []models.MappingService{}
	for _, s := range tpm.Services {
		svcs = append(svcs, models.MappingService(s))
	}

	return svcs, nil
}

func (r *Repo) GetDataproductsByMapping(ctx context.Context, service models.MappingService, limit, offset int) ([]*models.Dataproduct, error) {
	dataproducts := []*models.Dataproduct{}
	dps, err := r.querier.GetDataproductsByMapping(ctx, gensql.GetDataproductsByMappingParams{
		Service: string(service),
		Lim:     int32(limit),
		Offs:    int32(offset),
	})
	if err != nil {
		return nil, err
	}

	for _, entry := range dps {
		dataproducts = append(dataproducts, dataproductFromSQL(entry))
	}

	return dataproducts, nil
}

func (r *Repo) MapDataset(ctx context.Context, datasetID uuid.UUID, services []models.MappingService) error {
	svcs := []string{}
	for _, s := range services {
		svcs = append(svcs, string(s))
	}

	err := r.querier.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: datasetID,
		Services:  svcs,
	})
	if err != nil {
		return err
	}

	if contains(svcs, models.MappingServiceMetabase) {
		r.events.TriggerDataproductAddMetabaseMapping(ctx, datasetID)
	} else {
		r.events.TriggerDataproductRemoveMetabaseMapping(ctx, datasetID)
	}

	return nil
}

func contains(svcList []string, svc models.MappingService) bool {
	for _, s := range svcList {
		if s == string(svc) {
			return true
		}
	}
	return false
}
