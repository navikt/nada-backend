-- name: GetDataset :one
SELECT
  *
FROM
  datasets
WHERE
  id = @id;

-- name: GetDatasets :many
SELECT
  *
FROM
  datasets
ORDER BY
  last_modified DESC
LIMIT
  sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetDatasetsByIDs :many
SELECT
  *
FROM
  datasets
WHERE
  id = ANY (@ids :: uuid [])
ORDER BY
  last_modified DESC;

-- name: GetDatasetsByGroups :many
SELECT
  *
FROM
  datasets
WHERE
  "group" = ANY (@groups :: text [])
ORDER BY
  last_modified DESC;

-- name: GetDatasetsByUserAccess :many
SELECT
  *
FROM
  datasets
WHERE
  id = ANY (
    SELECT
      dataset_id
    FROM
      dataset_access
    WHERE
      "subject" = LOWER(@id)
      AND revoked IS NULL
      AND (
        expires > NOW()
        OR expires IS NULL
      )
  )
ORDER BY
  last_modified DESC;

-- name: GetDatasetsInDataproduct :many
SELECT
  *
FROM
  datasets
WHERE
  dataproduct_id = @dataproduct_id;

-- name: DeleteDataset :exec
DELETE FROM
  datasets
WHERE
  id = @id;

-- name: CreateDataset :one
INSERT INTO
  datasets (
    "dataproduct_id",
    "name",
    "description",
    "pii",
    "type",
    "slug",
    "repo",
    "keywords",
    "anonymisation_description",
    "target_user"
  )
VALUES
  (
    @dataproduct_id,
    @name,
    @description,
    @pii,
    @type,
    @slug,
    @repo,
    @keywords,
    @anonymisation_description,
    @target_user
  ) RETURNING *;

-- name: UpdateDataset :one
UPDATE
  datasets
SET
  "name" = @name,
  "description" = @description,
  "pii" = @pii,
  "slug" = @slug,
  "repo" = @repo,
  "keywords" = @keywords,
  "dataproduct_id" = @dataproduct_id,
  "anonymisation_description" = @anonymisation_description,
  "target_user" = @target_user
WHERE
  id = @id RETURNING *;

-- name: GetBigqueryDatasource :one
SELECT
  *
FROM
  datasource_bigquery
WHERE
  dataset_id = @dataset_id
  AND is_reference = @is_reference;

-- name: GetBigqueryDatasources :many
SELECT
  *
FROM
  datasource_bigquery;

-- name: CreateBigqueryDatasource :one
INSERT INTO
  datasource_bigquery (
    "dataset_id",
    "project_id",
    "dataset",
    "table_name",
    "schema",
    "last_modified",
    "created",
    "expires",
    "table_type",
    "pii_tags",
    "pseudo_columns",
    "is_reference"
  )
VALUES
  (
    @dataset_id,
    @project_id,
    @dataset,
    @table_name,
    @schema,
    @last_modified,
    @created,
    @expires,
    @table_type,
    @pii_tags,
    @pseudo_columns,
    @is_reference
  ) RETURNING *;

-- name: UpdateBigqueryDatasourceSchema :exec
UPDATE
  datasource_bigquery
SET
  "schema" = @schema,
  "last_modified" = @last_modified,
  "expires" = @expires,
  "description" = @description,
  "missing_since" = null,
  "pseudo_columns" = CASE
    WHEN @pseudo_columns IS NOT NULL THEN @pseudo_columns
    ELSE "pseudo_columns"
  END
WHERE
  dataset_id = @dataset_id;

-- name: UpdateBigqueryDatasource :exec
UPDATE
  datasource_bigquery
SET
  "pii_tags" = @pii_tags,
  "pseudo_columns" = @pseudo_columns
WHERE
  dataset_id = @dataset_id;

-- name: UpdateBigqueryDatasourceMissing :exec
UPDATE
  datasource_bigquery
SET
  "missing_since" = NOW()
WHERE
  dataset_id = @dataset_id;

-- name: DatasetsByMetabase :many
SELECT
  *
FROM
  datasets
WHERE
  id IN (
    SELECT
      dataset_id
    FROM
      metabase_metadata
    WHERE
      "deleted_at" IS NULL
  )
ORDER BY
  last_modified DESC
LIMIT
  @lim OFFSET @offs;

-- name: ReplaceDatasetsTag :exec
UPDATE
  datasets
SET
  "keywords" = array_replace(keywords, @tag_to_replace, @tag_updated);

-- name: GetAccessiblePseudoDatasetsByUser :many
WITH owned_dp AS(
  SELECT
    dp.id
  FROM
    dataproducts dp
  WHERE
    dp.group = ANY(@owner_subjects :: text [])
)
SELECT
  included_ds.id AS dataset_id,
  included_ds.name AS name,
  sbq.project_id AS bq_project_id,
  sbq.dataset AS bq_dataset_id,
  sbq.table_name AS bq_table_id,
  sbq.id AS bq_datasource_id
FROM
  (
    (
      SELECT
        ds.id AS id,
        ds.name AS name,
        ds.dataproduct_id AS dataproduct_id
      FROM
        datasets ds
        INNER JOIN dataset_access da ON ds.id = da.dataset_id
      WHERE
        da.subject = ANY(@access_subjects :: text [])
        AND (
          da.revoked IS NULL
          AND(
            da.expires IS NULL
            OR da.expires > CURRENT_TIMESTAMP
          )
        )
      GROUP BY
        ds.id
    )
    UNION
    (
      SELECT
        ds.id AS id,
        ds.name AS name,
        ds.dataproduct_id AS dataproduct_id
      FROM
        datasets ds
        INNER JOIN owned_dp ON ds.dataproduct_id = owned_dp.id
    )
  ) AS included_ds
  INNER JOIN datasource_bigquery AS sbq ON included_ds.id = sbq.dataset_id
  AND sbq.is_reference = TRUE;

-- name: GetJoinableViewsForOwner :many
SELECT
  jv.id AS id,
  jv.name AS name,
  jv.owner AS owner,
  jv.created AS created,
  jv.expires AS expires,
  bq.project_id AS project_id,
  bq.dataset AS dataset_id,
  bq.table_name AS table_id
FROM
  (
    joinable_views jv
    INNER JOIN (
      joinable_views_datasource jds
      INNER JOIN datasource_bigquery bq ON jds.datasource_id = bq.id
    ) ON jv.id = jds.joinable_view_id
  )
WHERE
  jv.owner = @owner
  AND (
    jv.expires IS NULL
    OR jv.expires > NOW()
  );

-- name: CreateJoinableViews :one
INSERT INTO
  joinable_views ("name", "owner", "created", "expires")
VALUES
  (@name, @owner, @created, @expires) RETURNING *;

-- name: CreateJoinableViewsDatasource :one
INSERT INTO
  joinable_views_datasource ("joinable_view_id", "datasource_id")
VALUES
  (@joinable_view_id, @datasource_id) RETURNING *;

-- name: GetJoinableViewsForReferenceAndUser :many
SELECT
  a.id as id,
  a.name as dataset
FROM
  joinable_views a
  JOIN joinable_views_datasource b ON a.id = b.joinable_view_id
  JOIN datasource_bigquery c ON b.datasource_id = c.id
WHERE
  owner = @owner
  AND c.dataset_id = @pseudo_dataset_id;

-- name: GetJoinableViewsWithReference :many
SELECT
  a.owner as owner,
  a.id as joinable_view_id,
  a.name as joinable_view_dataset,
  c.dataset_id as pseudo_view_id,
  c.project_id as pseudo_project_id,
  c.dataset as pseudo_dataset,
  c.table_name as pseudo_table,
  a.expires as expires
FROM
  joinable_views a
  JOIN joinable_views_datasource b ON a.id = b.joinable_view_id
  JOIN datasource_bigquery c ON b.datasource_id = c.id
WHERE
  a.deleted IS NULL;

-- name: GetOwnerGroupOfDataset :one
SELECT
  d.group as group
FROM
  dataproducts d
WHERE
  d.id = (
    SELECT
      dataproduct_id
    FROM
      datasets ds
    WHERE
      ds.id = @dataset_id
  );

-- name: GetDatasetsForOwner :many
SELECT
  ds.*
FROM
  datasets ds
WHERE
  dataproduct_id IN (
    SELECT
      id
    FROM
      dataproducts dp
    WHERE
      dp.group = ANY(@groups :: text [])
  );

-- name: SetJoinableViewDeleted :exec
UPDATE
  joinable_views
SET
  deleted = NOW()
WHERE
  id = @id;