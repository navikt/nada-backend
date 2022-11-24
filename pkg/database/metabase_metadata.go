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
		DatasetID:  metadata.DatasetID,
		DatabaseID: int32(metadata.DatabaseID),
		PermissionGroupID: sql.NullInt32{
			Int32: int32(metadata.PermissionGroupID),
			Valid: metadata.PermissionGroupID > 0,
		},
		AadPremissionGroupID: sql.NullInt32{
			Int32: int32(metadata.AADPermissionGroupID),
			Valid: metadata.AADPermissionGroupID > 0,
		},
		CollectionID: sql.NullInt32{
			Int32: int32(metadata.CollectionID),
			Valid: metadata.CollectionID > 0,
		},
		SaEmail: metadata.SAEmail,
	})
}

func (r *Repo) GetMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, includeDeleted bool) (*models.MetabaseMetadata, error) {
	var meta gensql.MetabaseMetadatum
	var err error
	if includeDeleted {
		meta, err = r.querier.GetMetabaseMetadataWithDeleted(ctx, datasetID)
	} else {
		meta, err = r.querier.GetMetabaseMetadata(ctx, datasetID)
	}

	if err != nil {
		return nil, err
	}

	return mbMetadataFromSQL(meta), nil
}

func (r *Repo) GetAllMetabaseMetadata(ctx context.Context) ([]*models.MetabaseMetadata, error) {
	mbs, err := r.querier.GetAllMetabaseMetadata(ctx)
	if err != nil {
		return nil, err
	}

	mbMetas := make([]*models.MetabaseMetadata, len(mbs))
	for idx, meta := range mbs {
		mbMetas[idx] = mbMetadataFromSQL(meta)
	}

	return mbMetas, nil
}

func (r *Repo) SoftDeleteMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error {
	return r.querier.SoftDeleteMetabaseMetadata(ctx, dataproductID)
}

func (r *Repo) SetPermissionGroupMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, groupID int) error {
	return r.querier.SetPermissionGroupMetabaseMetadata(ctx, gensql.SetPermissionGroupMetabaseMetadataParams{
		ID:        sql.NullInt32{Valid: true, Int32: int32(groupID)},
		DatasetID: datasetID,
	})
}

func (r *Repo) RestoreMetabaseMetadata(ctx context.Context, dataproductID uuid.UUID) error {
	return r.querier.RestoreMetabaseMetadata(ctx, dataproductID)
}

func (r *Repo) DeleteMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	mapping, err := r.querier.GetDatasetMappings(ctx, datasetID)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	querier := r.querier.WithTx(tx)
	if err := querier.DeleteMetabaseMetadata(ctx, datasetID); err != nil {
		return err
	}
	err = querier.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: datasetID,
		Services:  mapping.Services,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			r.log.WithError(err).Error("Rolling back cleanup metabase metadata transaction")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func mbMetadataFromSQL(meta gensql.MetabaseMetadatum) *models.MetabaseMetadata {
	return &models.MetabaseMetadata{
		DatasetID:            meta.DatasetID,
		DatabaseID:           int(meta.DatabaseID),
		PermissionGroupID:    int(meta.PermissionGroupID.Int32),
		AADPermissionGroupID: int(meta.AadPremissionGroupID.Int32),
		CollectionID:         int(meta.CollectionID.Int32),
		SAEmail:              meta.SaEmail,
		DeletedAt:            nullTimeToPtr(meta.DeletedAt),
	}
}

func removeMetabaseMapping(mapping gensql.ThirdPartyMapping) gensql.ThirdPartyMapping {
	for i, m := range mapping.Services {
		if m == "metabase" {
			mapping.Services = append(mapping.Services[:i], mapping.Services[i+1:]...)
		}
	}
	return mapping
}
