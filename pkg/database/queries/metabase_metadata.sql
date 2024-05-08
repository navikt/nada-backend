-- name: CreateMetabaseMetadata :exec
INSERT INTO metabase_metadata (
    "dataset_id",
    "database_id",
    "permission_group_id",
    "collection_id",
    "sa_email",
    "deleted_at"
) VALUES (
    @dataset_id,
    @database_id,
    @permission_group_id,
    @collection_id,
    @sa_email,
    @deleted_at
);

-- name: SoftDeleteMetabaseMetadata :exec
UPDATE metabase_metadata
SET "deleted_at" = NOW()
WHERE dataset_id = @dataset_id;

-- name: RestoreMetabaseMetadata :exec
UPDATE metabase_metadata
SET "deleted_at" = null
WHERE dataset_id = @dataset_id;

-- name: SetPermissionGroupMetabaseMetadata :exec
UPDATE metabase_metadata
SET "permission_group_id" = @id
WHERE dataset_id = @dataset_id;

-- name: GetMetabaseMetadata :one
SELECT *
FROM metabase_metadata
WHERE "dataset_id" = @dataset_id AND "deleted_at" IS NULL;

-- name: GetAllMetabaseMetadata :many
SELECT *
FROM metabase_metadata;

-- name: GetMetabaseMetadataWithDeleted :one
SELECT *
FROM metabase_metadata
WHERE "dataset_id" = @dataset_id;

-- name: DeleteMetabaseMetadata :exec
DELETE 
FROM metabase_metadata
WHERE "dataset_id" = @dataset_id;

-- name: GetOpenMetabaseTablesInSameBigQueryDataset :many
WITH sources_in_same_dataset AS (
  SELECT * FROM datasource_bigquery 
  WHERE project_id = @project_id AND dataset = @dataset
)

SELECT table_name FROM sources_in_same_dataset sds
JOIN metabase_metadata mbm
ON mbm.dataset_id = sds.dataset_id
WHERE mbm.collection_id IS null;
