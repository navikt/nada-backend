-- name: GetDataset :one
SELECT *
FROM datasets
WHERE id = @id;

-- name: GetDatasets :many
SELECT *
FROM datasets
ORDER BY last_modified DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');


-- name: GetDatasetsByIDs :many
SELECT *
FROM datasets
WHERE id = ANY (@ids::uuid[])
ORDER BY last_modified DESC;

-- name: GetDatasetsByGroups :many
SELECT *
FROM datasets
WHERE "group" = ANY (@groups::text[])
ORDER BY last_modified DESC;

-- name: GetDatasetsByUserAccess :many
SELECT *
FROM datasets
WHERE id = ANY (SELECT dataset_id
                FROM dataset_access
                WHERE "subject" = LOWER(@id)
                  AND revoked IS NULL
                  AND (expires > NOW() OR expires IS NULL))
ORDER BY last_modified DESC;

-- name: GetDatasetsInDataproduct :many
SELECT *
FROM datasets
WHERE dataproduct_id = @dataproduct_id;

-- name: DeleteDataset :exec
DELETE
FROM datasets
WHERE id = @id;

-- name: CreateDataset :one
INSERT INTO datasets ("dataproduct_id",
                      "name",
                      "description",
                      "pii",
                      "type",
                      "slug",
                      "repo",
                      "keywords")
VALUES (@dataproduct_id,
        @name,
        @description,
        @pii,
        @type,
        @slug,
        @repo,
        @keywords)
RETURNING *;

-- name: UpdateDataset :one
UPDATE datasets
SET "name"              = @name,
    "description"       = @description,
    "pii"               = @pii,
    "slug"              = @slug,
    "repo"              = @repo,
    "keywords"          = @keywords,
    "dataproduct_id"    = @dataproduct_id
WHERE id = @id
RETURNING *;

-- name: GetBigqueryDatasource :one
SELECT *
FROM datasource_bigquery
WHERE dataset_id = @dataset_id;

-- name: GetBigqueryDatasources :many
SELECT *
FROM datasource_bigquery;

-- name: CreateBigqueryDatasource :one
INSERT INTO datasource_bigquery ("dataset_id",
                                 "project_id",
                                 "dataset",
                                 "table_name",
                                 "schema",
                                 "last_modified",
                                 "created",
                                 "expires",
                                 "table_type")
VALUES (@dataset_id,
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
    "expires"       = @expires,
    "description"   = @description
WHERE dataset_id = @dataset_id;

-- name: GetDatasetRequesters :many
SELECT "subject"
FROM dataset_requesters
WHERE dataset_id = @dataset_id;

-- name: CreateDatasetRequester :exec
INSERT INTO dataset_requesters (dataset_id, "subject")
VALUES (@dataset_id, LOWER(@subject));

-- name: DeleteDatasetRequester :exec
DELETE
FROM dataset_requesters
WHERE dataset_id = @dataset_id
  AND "subject" = LOWER(@subject);

-- name: DatasetKeywords :many
SELECT keyword::text, count(1) as "count"
FROM (
	SELECT unnest(keywords) as keyword
	FROM datasets
) s
WHERE true
AND CASE WHEN coalesce(TRIM(@keyword), '') = '' THEN true ELSE keyword ILIKE @keyword::text || '%' END
GROUP BY keyword
ORDER BY "count" DESC
LIMIT 15;

-- name: DatasetsByMetabase :many
SELECT *
FROM datasets
WHERE id IN (
	SELECT dataset_id
	FROM metabase_metadata
  WHERE "deleted_at" IS NULL
)
ORDER BY last_modified DESC
LIMIT @lim OFFSET @offs;
