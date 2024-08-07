package service

import (
	"context"

	"github.com/google/uuid"
)

type ThirdPartyMappingStorage interface {
	MapDataset(ctx context.Context, datasetID uuid.UUID, Services []string) error
	GetAddMetabaseDatasetMappings(ctx context.Context) ([]uuid.UUID, error)
	GetRemoveMetabaseDatasetMappings(ctx context.Context) ([]uuid.UUID, error)
}
