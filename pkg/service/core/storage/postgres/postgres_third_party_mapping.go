package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
)

type thirdPartyMappingStorage struct {
	db *database.Repo
}

func (s *thirdPartyMappingStorage) MapDataset(ctx context.Context, datasetID string, services []string) error {
	const op errs.Op = "thirdPartyMappingStorage.MapDataset"

	err := s.db.Querier.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: uuid.MustParse(datasetID),
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
