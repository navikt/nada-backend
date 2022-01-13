-- name: CreateMetabaseMetadata :exec
INSERT INTO metabase_metadata (
    "dataproduct_id",
    "database_id",
    "permission_group_id",
    "sa_email"
) VALUES (
    @dataproduct_id,
    @database_id,
    @permission_group_id,
    @sa_email
);

-- name: GetMetabaseMetadata :one
SELECT *
FROM metabase_metadata
WHERE "dataproduct_id" = @dataproduct_id;

-- name: DeleteMetabaseMetadata :exec
DELETE 
FROM metabase_metadata
WHERE "dataproduct_id" = @dataproduct_id;
