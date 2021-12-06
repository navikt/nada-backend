-- name: CreateMetabaseMetadata :exec
INSERT INTO metabase_metadata (
    "dataproduct_id",
    "database_id",
    "permission_group_id"
) VALUES (
    @dataproduct_id,
    @database_id,
    @permission_group_id
);

-- name: GetMetabaseMetadata :one
SELECT *
FROM metabase_metadata
WHERE "dataproduct_id" = @dataproduct_id;
