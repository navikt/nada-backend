package service

import (
	"context"
)

type ThirdPartyMappingStorage interface {
	MapDataset(ctx context.Context, datasetID string, Services []string) error
}
