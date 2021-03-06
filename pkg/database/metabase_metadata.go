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
		CollectionID: sql.NullInt32{
			Int32: int32(metadata.CollectionID),
			Valid: metadata.CollectionID > 0,
		},
		SaEmail: metadata.SAEmail,
	})
}

func (r *Repo) GetMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID, includeDeleted bool) (*models.MetabaseMetadata, error) {
	var meta gensql.MetabaseMetadatum
	var err error
	if includeDeleted {
		meta, err = r.querier.GetMetabaseMetadataWithDeleted(ctx, dataproductID)
	} else {
		meta, err = r.querier.GetMetabaseMetadata(ctx, dataproductID)
	}

	if err != nil {
		return nil, err
	}

	return &models.MetabaseMetadata{
		DataproductID:     meta.DataproductID,
		DatabaseID:        int(meta.DatabaseID),
		PermissionGroupID: int(meta.PermissionGroupID.Int32),
		CollectionID:      int(meta.CollectionID.Int32),
		SAEmail:           meta.SaEmail,
		DeletedAt:         nullTimeToPtr(meta.DeletedAt),
	}, nil
}

func (r *Repo) SoftDeleteMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error {
	return r.querier.SoftDeleteMetabaseMetadata(ctx, dataproductID)
}

func (r *Repo) SetPermissionGroupMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID, groupID int) error {
	return r.querier.SetPermissionGroupMetabaseMetadata(ctx, gensql.SetPermissionGroupMetabaseMetadataParams{
		ID:            sql.NullInt32{Valid: true, Int32: int32(groupID)},
		DataproductID: dataproductID,
	})
}

func (r *Repo) RestoreMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error {
	return r.querier.RestoreMetabaseMetadata(ctx, dataproductID)
}

func (r *Repo) DeleteMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error {
	return r.querier.DeleteMetabaseMetadata(ctx, dataproductID)
}
