package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
	"time"
)

type MetabaseMetadata gensql.MetabaseMetadatum

// FIXME: should define an interface that uses the subset of Querier methods that we need
// and reference that here
type metabaseStorage struct {
	db *database.Repo
}

// Ensure that we always implement the service.MetabaseStorage interface
var _ service.MetabaseStorage = &metabaseStorage{}

func NewMetabaseStorage(db *database.Repo) *metabaseStorage {
	return &metabaseStorage{
		db: db,
	}
}

func (s *metabaseStorage) SetPermissionGroupMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, groupID int) error {
	const op errs.Op = "postgres.SetPermissionGroupMetabaseMetadata"

	err := s.db.Querier.SetPermissionGroupMetabaseMetadata(ctx, gensql.SetPermissionGroupMetabaseMetadataParams{
		ID:        sql.NullInt32{Valid: true, Int32: int32(groupID)},
		DatasetID: datasetID,
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) CreateMetadata(ctx context.Context, metadata *service.MetabaseMetadata) error {
	const op errs.Op = "postgres.CreateMetadata"

	params := gensql.CreateMetabaseMetadataParams{
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
	}

	err := s.db.Querier.CreateMetabaseMetadata(ctx, params)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) GetMetadata(ctx context.Context, datasetID uuid.UUID, includeDeleted bool) (*service.MetabaseMetadata, error) {
	const op errs.Op = "postgres.GetMetadata"

	if includeDeleted {
		meta, err := s.db.Querier.GetMetabaseMetadataWithDeleted(ctx, datasetID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errs.E(errs.NotExist, op, err)
			}

			return nil, errs.E(errs.Database, op, err)
		}

		return ToLocal(meta).Convert(), nil
	}

	meta, err := s.db.Querier.GetMetabaseMetadata(ctx, datasetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, err)
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return ToLocal(meta).Convert(), nil
}

func (s *metabaseStorage) GetAllMetadata(ctx context.Context) ([]*service.MetabaseMetadata, error) {
	const op errs.Op = "postgres.GetAllMetadata"

	mbs, err := s.db.Querier.GetAllMetabaseMetadata(ctx)
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	mbMetas := make([]*service.MetabaseMetadata, len(mbs))
	for idx, meta := range mbs {
		mbMetas[idx] = ToLocal(meta).Convert()
	}

	return mbMetas, nil
}

func (s *metabaseStorage) GetOpenTablesInSameBigQueryDataset(ctx context.Context, projectID, dataset string) ([]string, error) {
	const op errs.Op = "postgres.GetOpenTablesInSameBigQueryDataset"

	tables, err := s.db.Querier.GetOpenMetabaseTablesInSameBigQueryDataset(ctx, gensql.GetOpenMetabaseTablesInSameBigQueryDatasetParams{
		ProjectID: projectID,
		Dataset:   dataset,
	})
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	return tables, nil
}

func (s *metabaseStorage) SetPermissionsGroup(ctx context.Context, datasetID uuid.UUID, groupID int) error {
	const op errs.Op = "postgres.SetPermissionsGroup"

	err := s.db.Querier.SetPermissionGroupMetabaseMetadata(ctx, gensql.SetPermissionGroupMetabaseMetadataParams{
		ID:        sql.NullInt32{Valid: true, Int32: int32(groupID)},
		DatasetID: datasetID,
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) SoftDeleteMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "postgres.SoftDeleteMetadata"

	err := s.db.Querier.SoftDeleteMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) RestoreMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "postgres.RestoreMetadata"

	err := s.db.Querier.RestoreMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) DeleteMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "postgres.DeleteMetadata"

	err := s.db.Querier.DeleteMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) DeleteRestrictedMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "postgres.DeleteRestrictedMetadata"

	mapping, err := s.db.Querier.GetDatasetMappings(ctx, datasetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errs.E(errs.NotExist, op, err)
		}

		return errs.E(errs.Database, op, err)
	}

	tx, err := s.db.GetDB().Begin()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}
	defer tx.Rollback()

	querier := s.db.Querier.WithTx(tx)
	err = querier.DeleteMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	err = querier.MapDataset(ctx, gensql.MapDatasetParams{
		DatasetID: datasetID,
		Services:  mapping.Services,
	})
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	err = tx.Commit()
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func ToLocal(m gensql.MetabaseMetadatum) MetabaseMetadata {
	return MetabaseMetadata(m)
}

func (m MetabaseMetadata) Convert() *service.MetabaseMetadata {
	return &service.MetabaseMetadata{
		DatasetID:         m.DatasetID,
		DatabaseID:        int(m.DatabaseID),
		PermissionGroupID: int(m.PermissionGroupID.Int32),
		CollectionID:      int(m.CollectionID.Int32),
		SAEmail:           m.SaEmail,
		DeletedAt:         nullTimeToPtr(m.DeletedAt),
	}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}

	return &nt.Time
}
