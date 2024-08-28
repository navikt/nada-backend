package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/navikt/nada-backend/pkg/database/gensql"
	"github.com/navikt/nada-backend/pkg/errs"
	"github.com/navikt/nada-backend/pkg/service"
)

type MetabaseMetadata gensql.MetabaseMetadatum

// Ensure that we always implement the service.MetabaseStorage interface
var _ service.MetabaseStorage = &metabaseStorage{}

// FIXME: should define an interface that uses the subset of Querier methods that we need
// and reference that here
type metabaseStorage struct {
	db *database.Repo
}

func (s *metabaseStorage) SetCollectionMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, collectionID int) (*service.MetabaseMetadata, error) {
	const op errs.Op = "metabaseStorage.SetCollectionMetabaseMetadata"

	meta, err := s.db.Querier.SetCollectionMetabaseMetadata(ctx, gensql.SetCollectionMetabaseMetadataParams{
		CollectionID: sql.NullInt32{Valid: true, Int32: int32(collectionID)},
		DatasetID:    datasetID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, fmt.Errorf("setting collection %v: %w", collectionID, err))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return ToLocal(meta).Convert(), nil
}

func (s *metabaseStorage) SetDatabaseMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, databaseID int) (*service.MetabaseMetadata, error) {
	const op errs.Op = "metabaseStorage.SetDatabaseMetabaseMetadata"

	meta, err := s.db.Querier.SetDatabaseMetabaseMetadata(ctx, gensql.SetDatabaseMetabaseMetadataParams{
		DatabaseID: sql.NullInt32{Valid: true, Int32: int32(databaseID)},
		DatasetID:  datasetID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, fmt.Errorf("setting database %v: %w", databaseID, err))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return ToLocal(meta).Convert(), nil
}

func (s *metabaseStorage) SetServiceAccountMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, saEmail string) (*service.MetabaseMetadata, error) {
	const op errs.Op = "metabaseStorage.SetServiceAccountMetabaseMetadata"

	meta, err := s.db.Querier.SetServiceAccountMetabaseMetadata(ctx, gensql.SetServiceAccountMetabaseMetadataParams{
		SaEmail:   saEmail,
		DatasetID: datasetID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, fmt.Errorf("setting sa_email %v: %w", saEmail, err))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return ToLocal(meta).Convert(), nil
}

func (s *metabaseStorage) SetSyncCompletedMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "metabaseStorage.SetSyncCompletedMetabaseMetadata"

	err := s.db.Querier.SetSyncCompletedMetabaseMetadata(ctx, datasetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errs.E(errs.NotExist, op, fmt.Errorf("setting sync completed: %w", err))
		}

		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) SetPermissionGroupMetabaseMetadata(ctx context.Context, datasetID uuid.UUID, groupID int) (*service.MetabaseMetadata, error) {
	const op errs.Op = "metabaseStorage.SetPermissionGroupMetabaseMetadata"

	meta, err := s.db.Querier.SetPermissionGroupMetabaseMetadata(ctx, gensql.SetPermissionGroupMetabaseMetadataParams{
		PermissionGroupID: sql.NullInt32{Valid: true, Int32: int32(groupID)},
		DatasetID:         datasetID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, fmt.Errorf("setting permissions group %v: %w", groupID, err))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return ToLocal(meta).Convert(), nil
}

func (s *metabaseStorage) CreateMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "metabaseStorage.CreateMetadata"

	err := s.db.Querier.CreateMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) GetMetadata(ctx context.Context, datasetID uuid.UUID, includeDeleted bool) (*service.MetabaseMetadata, error) {
	const op errs.Op = "metabaseStorage.GetMetadata"

	if includeDeleted {
		meta, err := s.db.Querier.GetMetabaseMetadataWithDeleted(ctx, datasetID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errs.E(errs.NotExist, op, fmt.Errorf("getting dataset %v: %w", datasetID, err))
			}

			return nil, errs.E(errs.Database, op, err)
		}

		return ToLocal(meta).Convert(), nil
	}

	meta, err := s.db.Querier.GetMetabaseMetadata(ctx, datasetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.E(errs.NotExist, op, fmt.Errorf("getting dataset %v: %w", datasetID, err))
		}

		return nil, errs.E(errs.Database, op, err)
	}

	return ToLocal(meta).Convert(), nil
}

func (s *metabaseStorage) GetAllMetadata(ctx context.Context) ([]*service.MetabaseMetadata, error) {
	const op errs.Op = "metabaseStorage.GetAllMetadata"

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
	const op errs.Op = "metabaseStorage.GetOpenTablesInSameBigQueryDataset"

	tables, err := s.db.Querier.GetOpenMetabaseTablesInSameBigQueryDataset(ctx, gensql.GetOpenMetabaseTablesInSameBigQueryDatasetParams{
		ProjectID: projectID,
		Dataset:   dataset,
	})
	if err != nil {
		return nil, errs.E(errs.Database, op, err)
	}

	return tables, nil
}

func (s *metabaseStorage) SoftDeleteMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "metabaseStorage.SoftDeleteMetadata"

	err := s.db.Querier.SoftDeleteMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) RestoreMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "metabaseStorage.RestoreMetadata"

	err := s.db.Querier.RestoreMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) DeleteMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "metabaseStorage.DeleteMetadata"

	err := s.db.Querier.DeleteMetabaseMetadata(ctx, datasetID)
	if err != nil {
		return errs.E(errs.Database, op, err)
	}

	return nil
}

func (s *metabaseStorage) DeleteRestrictedMetadata(ctx context.Context, datasetID uuid.UUID) error {
	const op errs.Op = "metabaseStorage.DeleteRestrictedMetadata"

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
		DatabaseID:        nullInt32ToIntPtr(m.DatabaseID),
		PermissionGroupID: nullInt32ToIntPtr(m.PermissionGroupID),
		CollectionID:      nullInt32ToIntPtr(m.CollectionID),
		SAEmail:           m.SaEmail,
		DeletedAt:         nullTimeToPtr(m.DeletedAt),
		SyncCompleted:     nullTimeToPtr(m.SyncCompleted),
	}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}

	return &nt.Time
}

func NewMetabaseStorage(db *database.Repo) *metabaseStorage {
	return &metabaseStorage{
		db: db,
	}
}
