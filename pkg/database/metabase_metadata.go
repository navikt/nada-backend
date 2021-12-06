package database

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/graph/models"
)

func (r *Repo) CreateMetabaseMetadata(ctx context.Context, metadata models.MetabaseMetadata) error {
	return r.querier.CreateMetabaseMetadata(ctx, gensql.CreateMetabaseMetadataParams{
		DataproductID: metadata.DataproductID,
		DatabaseID:    int32(metadata.DatabaseID),
		PermissionGroupID: sql.NullInt32{
			Int32: int32(metadata.PermissionGroupID),
			Valid: metadata.PermissionGroupID > 0,
		},
	})
}

func (r *Repo) GetMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) (models.MetabaseMetadata, error) {
	meta, err := r.querier.GetMetabaseMetadata(ctx, dataproductID)
	if err != nil {
		return models.MetabaseMetadata{}, err
	}

	return models.MetabaseMetadata{
		DataproductID:     meta.DataproductID,
		DatabaseID:        int(meta.DatabaseID),
		PermissionGroupID: int(meta.PermissionGroupID.Int32),
	}, nil
}
