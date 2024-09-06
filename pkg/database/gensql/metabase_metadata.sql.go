// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: metabase_metadata.sql

package gensql

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const createMetabaseMetadata = `-- name: CreateMetabaseMetadata :exec
INSERT INTO metabase_metadata (
    "dataset_id"
) VALUES (
    $1
)
`

func (q *Queries) CreateMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, createMetabaseMetadata, datasetID)
	return err
}

const deleteMetabaseMetadata = `-- name: DeleteMetabaseMetadata :exec
DELETE 
FROM metabase_metadata
WHERE "dataset_id" = $1
`

func (q *Queries) DeleteMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteMetabaseMetadata, datasetID)
	return err
}

const getAllMetabaseMetadata = `-- name: GetAllMetabaseMetadata :many
SELECT database_id, permission_group_id, sa_email, collection_id, deleted_at, dataset_id, sync_completed
FROM metabase_metadata
`

func (q *Queries) GetAllMetabaseMetadata(ctx context.Context) ([]MetabaseMetadatum, error) {
	rows, err := q.db.QueryContext(ctx, getAllMetabaseMetadata)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MetabaseMetadatum{}
	for rows.Next() {
		var i MetabaseMetadatum
		if err := rows.Scan(
			&i.DatabaseID,
			&i.PermissionGroupID,
			&i.SaEmail,
			&i.CollectionID,
			&i.DeletedAt,
			&i.DatasetID,
			&i.SyncCompleted,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getMetabaseMetadata = `-- name: GetMetabaseMetadata :one
SELECT database_id, permission_group_id, sa_email, collection_id, deleted_at, dataset_id, sync_completed
FROM metabase_metadata
WHERE "dataset_id" = $1 AND "deleted_at" IS NULL
`

func (q *Queries) GetMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) (MetabaseMetadatum, error) {
	row := q.db.QueryRowContext(ctx, getMetabaseMetadata, datasetID)
	var i MetabaseMetadatum
	err := row.Scan(
		&i.DatabaseID,
		&i.PermissionGroupID,
		&i.SaEmail,
		&i.CollectionID,
		&i.DeletedAt,
		&i.DatasetID,
		&i.SyncCompleted,
	)
	return i, err
}

const getMetabaseMetadataWithDeleted = `-- name: GetMetabaseMetadataWithDeleted :one
SELECT database_id, permission_group_id, sa_email, collection_id, deleted_at, dataset_id, sync_completed
FROM metabase_metadata
WHERE "dataset_id" = $1
`

func (q *Queries) GetMetabaseMetadataWithDeleted(ctx context.Context, datasetID uuid.UUID) (MetabaseMetadatum, error) {
	row := q.db.QueryRowContext(ctx, getMetabaseMetadataWithDeleted, datasetID)
	var i MetabaseMetadatum
	err := row.Scan(
		&i.DatabaseID,
		&i.PermissionGroupID,
		&i.SaEmail,
		&i.CollectionID,
		&i.DeletedAt,
		&i.DatasetID,
		&i.SyncCompleted,
	)
	return i, err
}

const getOpenMetabaseTablesInSameBigQueryDataset = `-- name: GetOpenMetabaseTablesInSameBigQueryDataset :many
WITH sources_in_same_dataset AS (
  SELECT dataset_id, project_id, dataset, table_name, schema, last_modified, created, expires, table_type, description, pii_tags, missing_since, id, is_reference, pseudo_columns, deleted FROM datasource_bigquery 
  WHERE project_id = $1 AND dataset = $2
)

SELECT table_name FROM sources_in_same_dataset sds
JOIN metabase_metadata mbm
ON mbm.dataset_id = sds.dataset_id
WHERE mbm.collection_id IS null
`

type GetOpenMetabaseTablesInSameBigQueryDatasetParams struct {
	ProjectID string
	Dataset   string
}

func (q *Queries) GetOpenMetabaseTablesInSameBigQueryDataset(ctx context.Context, arg GetOpenMetabaseTablesInSameBigQueryDatasetParams) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getOpenMetabaseTablesInSameBigQueryDataset, arg.ProjectID, arg.Dataset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var table_name string
		if err := rows.Scan(&table_name); err != nil {
			return nil, err
		}
		items = append(items, table_name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const restoreMetabaseMetadata = `-- name: RestoreMetabaseMetadata :exec
UPDATE metabase_metadata
SET "deleted_at" = null
WHERE dataset_id = $1
`

func (q *Queries) RestoreMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, restoreMetabaseMetadata, datasetID)
	return err
}

const setCollectionMetabaseMetadata = `-- name: SetCollectionMetabaseMetadata :one
UPDATE metabase_metadata
SET "collection_id" = $1
WHERE dataset_id = $2
RETURNING database_id, permission_group_id, sa_email, collection_id, deleted_at, dataset_id, sync_completed
`

type SetCollectionMetabaseMetadataParams struct {
	CollectionID sql.NullInt32
	DatasetID    uuid.UUID
}

func (q *Queries) SetCollectionMetabaseMetadata(ctx context.Context, arg SetCollectionMetabaseMetadataParams) (MetabaseMetadatum, error) {
	row := q.db.QueryRowContext(ctx, setCollectionMetabaseMetadata, arg.CollectionID, arg.DatasetID)
	var i MetabaseMetadatum
	err := row.Scan(
		&i.DatabaseID,
		&i.PermissionGroupID,
		&i.SaEmail,
		&i.CollectionID,
		&i.DeletedAt,
		&i.DatasetID,
		&i.SyncCompleted,
	)
	return i, err
}

const setDatabaseMetabaseMetadata = `-- name: SetDatabaseMetabaseMetadata :one
UPDATE metabase_metadata
SET "database_id" = $1
WHERE dataset_id = $2
RETURNING database_id, permission_group_id, sa_email, collection_id, deleted_at, dataset_id, sync_completed
`

type SetDatabaseMetabaseMetadataParams struct {
	DatabaseID sql.NullInt32
	DatasetID  uuid.UUID
}

func (q *Queries) SetDatabaseMetabaseMetadata(ctx context.Context, arg SetDatabaseMetabaseMetadataParams) (MetabaseMetadatum, error) {
	row := q.db.QueryRowContext(ctx, setDatabaseMetabaseMetadata, arg.DatabaseID, arg.DatasetID)
	var i MetabaseMetadatum
	err := row.Scan(
		&i.DatabaseID,
		&i.PermissionGroupID,
		&i.SaEmail,
		&i.CollectionID,
		&i.DeletedAt,
		&i.DatasetID,
		&i.SyncCompleted,
	)
	return i, err
}

const setPermissionGroupMetabaseMetadata = `-- name: SetPermissionGroupMetabaseMetadata :one
UPDATE metabase_metadata
SET "permission_group_id" = $1
WHERE dataset_id = $2
RETURNING database_id, permission_group_id, sa_email, collection_id, deleted_at, dataset_id, sync_completed
`

type SetPermissionGroupMetabaseMetadataParams struct {
	PermissionGroupID sql.NullInt32
	DatasetID         uuid.UUID
}

func (q *Queries) SetPermissionGroupMetabaseMetadata(ctx context.Context, arg SetPermissionGroupMetabaseMetadataParams) (MetabaseMetadatum, error) {
	row := q.db.QueryRowContext(ctx, setPermissionGroupMetabaseMetadata, arg.PermissionGroupID, arg.DatasetID)
	var i MetabaseMetadatum
	err := row.Scan(
		&i.DatabaseID,
		&i.PermissionGroupID,
		&i.SaEmail,
		&i.CollectionID,
		&i.DeletedAt,
		&i.DatasetID,
		&i.SyncCompleted,
	)
	return i, err
}

const setServiceAccountMetabaseMetadata = `-- name: SetServiceAccountMetabaseMetadata :one
UPDATE metabase_metadata
SET "sa_email" = $1
WHERE dataset_id = $2
RETURNING database_id, permission_group_id, sa_email, collection_id, deleted_at, dataset_id, sync_completed
`

type SetServiceAccountMetabaseMetadataParams struct {
	SaEmail   string
	DatasetID uuid.UUID
}

func (q *Queries) SetServiceAccountMetabaseMetadata(ctx context.Context, arg SetServiceAccountMetabaseMetadataParams) (MetabaseMetadatum, error) {
	row := q.db.QueryRowContext(ctx, setServiceAccountMetabaseMetadata, arg.SaEmail, arg.DatasetID)
	var i MetabaseMetadatum
	err := row.Scan(
		&i.DatabaseID,
		&i.PermissionGroupID,
		&i.SaEmail,
		&i.CollectionID,
		&i.DeletedAt,
		&i.DatasetID,
		&i.SyncCompleted,
	)
	return i, err
}

const setSyncCompletedMetabaseMetadata = `-- name: SetSyncCompletedMetabaseMetadata :exec
UPDATE metabase_metadata
SET "sync_completed" = NOW()
WHERE dataset_id = $1
`

func (q *Queries) SetSyncCompletedMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, setSyncCompletedMetabaseMetadata, datasetID)
	return err
}

const softDeleteMetabaseMetadata = `-- name: SoftDeleteMetabaseMetadata :exec
UPDATE metabase_metadata
SET "deleted_at" = NOW()
WHERE dataset_id = $1
`

func (q *Queries) SoftDeleteMetabaseMetadata(ctx context.Context, datasetID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, softDeleteMetabaseMetadata, datasetID)
	return err
}
