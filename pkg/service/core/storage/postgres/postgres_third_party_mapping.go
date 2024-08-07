package postgres

import (
	"context"

	"github.com/navikt/nada-backend/pkg/service"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
)

var _ service.ThirdPartyMappingStorage = &thirdPartyMappingStorage{}

type thirdPartyMappingStorage struct {
	db *database.Repo
}

func (s *thirdPartyMappingStorage) GetRemoveMetabaseDatasetMappings(ctx context.Context) ([]uuid.UUID, error) {
	const op errs.Op = "thirdPartyMappingStorage.GetRemoveMetabaseDatasetMappings"

	datasetIDs, err := s.db.Querier.GetRemoveMetabaseDatasetMappings(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	return datasetIDs, nil
}

func (s *thirdPartyMappingStorage) GetAddMetabaseDatasetMappings(ctx context.Context) ([]uuid.UUID, error) {
	const op errs.Op = "thirdPartyMappingStorage.GetAddMetabaseDatasetMappings"

	datasetIDs, err := s.db.Querier.GetAddMetabaseDatasetMappings(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	return datasetIDs, nil
}

func (s *thirdPartyMappingStorage) MapDataset(ctx context.Context, datasetID uuid.UUID, services []string) error {
	const op errs.Op = "thirdPartyMappingStorage.MapDataset"

	err := s.db.Querier.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: datasetID,
		Services:  services,
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func NewThirdPartyMappingStorage(db *database.Repo) *thirdPartyMappingStorage {
	return &thirdPartyMappingStorage{
		db: db,
	}
}
