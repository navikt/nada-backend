package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type MappingService string

func GetDatasetMappings(ctx context.Context, datasetID uuid.UUID) ([]MappingService, error) {
	tpm, err := queries.GetDatasetMappings(ctx, datasetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []MappingService{}, nil
		}
		return nil, err
	}

	svcs := []MappingService{}
	for _, s := range tpm.Services {
		svcs = append(svcs, MappingService(s))
	}

	return svcs, nil
}
