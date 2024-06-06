package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database/gensql"
)

type MetabaseMetadata struct {
	DatasetID         uuid.UUID
	DatabaseID        int
	PermissionGroupID int
	CollectionID      int
	SAEmail           string
	DeletedAt         *time.Time
}

func CreateMetabaseMetadata(ctx context.Context, metadata MetabaseMetadata) error {
	return queries.CreateMetabaseMetadata(ctx, gensql.CreateMetabaseMetadataParams{
		DatasetID:  metadata.DatasetID,
		DatabaseID: int32(metadata.DatabaseID),
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

func GetMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, includeDeleted bool) (*MetabaseMetadata, error) {
	var meta gensql.MetabaseMetadatum
	var err error
	if includeDeleted {
		meta, err = queries.GetMetabaseMetadataWithDeleted(ctx, datasetID)
	} else {
		meta, err = queries.GetMetabaseMetadata(ctx, datasetID)
	}

	if err != nil {
		return nil, err
	}

	return mbMetadataFromSQL(meta), nil
}

func GetAllMetabaseMetadata(ctx context.Context) ([]*MetabaseMetadata, error) {
	mbs, err := queries.GetAllMetabaseMetadata(ctx)
	if err != nil {
		return nil, err
	}

	mbMetas := make([]*MetabaseMetadata, len(mbs))
	for idx, meta := range mbs {
		mbMetas[idx] = mbMetadataFromSQL(meta)
	}

	return mbMetas, nil
}

func GetOpenMetabaseTablesInSameBigQueryDataset(ctx context.Context, projectID, dataset string) ([]string, error) {
	return queries.GetOpenMetabaseTablesInSameBigQueryDataset(ctx, gensql.GetOpenMetabaseTablesInSameBigQueryDatasetParams{
		ProjectID: projectID,
		Dataset:   dataset,
	})
}

func SetPermissionGroupMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, groupID int) error {
	return queries.SetPermissionGroupMetabaseMetadata(ctx, gensql.SetPermissionGroupMetabaseMetadataParams{
		ID:        sql.NullInt32{Valid: true, Int32: int32(groupID)},
		DatasetID: datasetID,
	})
}

func SoftDeleteMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	return queries.SoftDeleteMetabaseMetadata(ctx, datasetID)
}

func RestoreMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	return queries.RestoreMetabaseMetadata(ctx, datasetID)
}

func DeleteMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	return queries.DeleteMetabaseMetadata(ctx, datasetID)
}

func DeleteRestrictedMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	mapping, err := queries.GetDatasetMappings(ctx, datasetID)
	if err != nil {
		return err
	}

	tx, err := sqldb.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	querier := queries.WithTx(tx)
	if err := querier.DeleteMetabaseMetadata(ctx, datasetID); err != nil {
		return err
	}
	err = querier.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: datasetID,
		Services:  mapping.Services,
	})
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func mbMetadataFromSQL(meta gensql.MetabaseMetadatum) *MetabaseMetadata {
	return &MetabaseMetadata{
		DatasetID:         meta.DatasetID,
		DatabaseID:        int(meta.DatabaseID),
		PermissionGroupID: int(meta.PermissionGroupID.Int32),
		CollectionID:      int(meta.CollectionID.Int32),
		SAEmail:           meta.SaEmail,
		DeletedAt:         nullTimeToPtr(meta.DeletedAt),
	}
}
