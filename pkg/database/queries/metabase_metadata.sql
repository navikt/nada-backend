-- name: CreateMetabaseMetadata :exec
INSERT INTO metabase_metadata (
    "dataproduct_id",
    "database_id",
    "permission_group_id",
    "collection_id",
    "sa_email",
    "deleted_at"
) VALUES (
    @dataproduct_id,
    @database_id,
    @permission_group_id,
    @collection_id,
    @sa_email,
    @deleted_at
);

-- name: SoftDeleteMetabaseMetadata :exec
UPDATE metabase_metadata
SET "deleted_at" = NOW()
WHERE dataproduct_id = @dataproduct_id;

-- name: RestoreMetabaseMetadata :exec
UPDATE metabase_metadata
SET "deleted_at" = null
WHERE dataproduct_id = @dataproduct_id;

-- name: SetPermissionGroupMetabaseMetadata :exec
UPDATE metabase_metadata
SET "permission_group_id" = @id
WHERE dataproduct_id = @dataproduct_id;

-- name: GetMetabaseMetadata :one
SELECT *
FROM metabase_metadata
WHERE "dataproduct_id" = @dataproduct_id AND "deleted_at" IS NULL;

-- name: GetMetabaseMetadataWithDeleted :one
SELECT *
FROM metabase_metadata
WHERE "dataproduct_id" = @dataproduct_id;

-- name: DeleteMetabaseMetadata :exec
DELETE 
FROM metabase_metadata
WHERE "dataproduct_id" = @dataproduct_id;
