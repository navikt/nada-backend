-- name: GetDataproduct :one
SELECT *
FROM dataproducts
WHERE id = @id;

-- name: GetDataproducts :many
SELECT *
FROM dataproducts
ORDER BY last_modified DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');


-- name: GetDataproductsByIDs :many
SELECT *
FROM dataproducts
WHERE id = ANY (@ids::uuid[])
ORDER BY last_modified DESC;

-- name: GetDataproductsByGroups :many
SELECT *
FROM dataproducts
WHERE "group" = ANY (@groups::text[])
ORDER BY last_modified DESC;

-- name: GetDataproductsByUserAccess :many
SELECT *
FROM dataproducts
WHERE id = ANY (SELECT dataproduct_id
                FROM dataproduct_access
                WHERE "subject" = LOWER(@id)
                  AND revoked IS NULL
                  AND (expires > NOW() OR expires IS NULL))
ORDER BY last_modified DESC;

-- name: DeleteDataproduct :exec
DELETE
FROM dataproducts
WHERE id = @id;

-- name: CreateDataproduct :one
INSERT INTO dataproducts ("name",
                          "description",
                          "pii",
                          "type",
                          "group",
                          "teamkatalogen_url",
                          "slug",
                          "repo",
                          "keywords")
VALUES (@name,
        @description,
        @pii,
        @type,
        @owner_group,
        @owner_teamkatalogen_url,
        @slug,
        @repo,
        @keywords)
RETURNING *;

-- name: UpdateDataproduct :one
UPDATE dataproducts
SET "name"              = @name,
    "description"       = @description,
    "pii"               = @pii,
    "slug"              = @slug,
    "repo"              = @repo,
    "teamkatalogen_url" = @owner_teamkatalogen_url,
    "keywords"          = @keywords
WHERE id = @id
RETURNING *;

-- name: GetBigqueryDatasource :one
SELECT *
FROM datasource_bigquery
WHERE dataproduct_id = @dataproduct_id;

-- name: GetBigqueryDatasources :many
SELECT *
FROM datasource_bigquery;

-- name: CreateBigqueryDatasource :one
INSERT INTO datasource_bigquery ("dataproduct_id",
                                 "project_id",
                                 "dataset",
                                 "table_name",
                                 "schema",
                                 "last_modified",
                                 "created",
                                 "expires",
                                 "table_type")
VALUES (@dataproduct_id,
        @project_id,
        @dataset,
        @table_name,
        @schema,
        @last_modified,
        @created,
        @expires,
        @table_type)
RETURNING *;

-- name: UpdateBigqueryDatasourceSchema :exec
UPDATE datasource_bigquery
SET "schema"        = @schema,
    "last_modified" = @last_modified,
    "expires"       = @expires
WHERE dataproduct_id = @dataproduct_id;

-- name: GetDataproductRequesters :many
SELECT "subject"
FROM dataproduct_requesters
WHERE dataproduct_id = @dataproduct_id;

-- name: CreateDataproductRequester :exec
INSERT INTO dataproduct_requesters (dataproduct_id, "subject")
VALUES (@dataproduct_id, LOWER(@subject));

-- name: DeleteDataproductRequester :exec
DELETE
FROM dataproduct_requesters
WHERE dataproduct_id = @dataproduct_id
  AND "subject" = LOWER(@subject);
