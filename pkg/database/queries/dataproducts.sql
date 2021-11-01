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
SELECT * FROM dataproducts WHERE id = ANY(@ids::uuid[]) ORDER BY last_modified DESC;

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
                          "slug",
                          "repo",
                          "keywords")
VALUES (@name,
        @description,
        @pii,
        @type,
        @owner_group,
        @slug,
        @repo,
        @keywords)
RETURNING *;

-- name: UpdateDataproduct :one
UPDATE dataproducts
SET "name"        = @name,
    "description" = @description,
    "pii"         = @pii,
    "slug"        = @slug,
    "repo"        = @repo,
    "keywords"    = @keywords
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
SET "schema" = @schema, "last_modified" = @last_modified, "expires" = @expires
WHERE dataproduct_id = @dataproduct_id;

-- name: GetDataproductRequesters :many
SELECT "subject"
FROM dataproduct_requesters
WHERE dataproduct_id = @dataproduct_id;

-- name: CreateDataproductRequester :exec
INSERT INTO dataproduct_requesters (dataproduct_id, "subject")
VALUES (@dataproduct_id, @subject);

-- name: DeleteDataproductRequester :exec
DELETE FROM dataproduct_requesters 
WHERE dataproduct_id = @dataproduct_id
AND "subject" = @subject;
