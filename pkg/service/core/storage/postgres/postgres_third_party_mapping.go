package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type thirdPartyMappingStorage struct {
	db *database.Repo
}

func (s *thirdPartyMappingStorage) MapDataset(ctx context.Context, datasetID string, services []string) error {
	err := s.db.Querier.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: uuid.MustParse(datasetID),
		Services:  services,
	})
	if err != nil {
		return fmt.Errorf("mapping dataset: %w", err)
	}

	return nil
}

func NewThirdPartyMappingStorage(db *database.Repo) *thirdPartyMappingStorage {
	return &thirdPartyMappingStorage{
		db: db,
	}
}
