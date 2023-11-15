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
    a.deleted IS NULL AND b.deleted IS NULL;

-- name: SetJoinableViewDeleted :exec
UPDATE
    joinable_views
SET
    deleted = NOW()
WHERE
    id = @id;

-- name: GetJoinableViewWithDataset :many
SELECT
    dsrc.project_id as bq_project,
    dsrc.dataset as bq_dataset,
    dsrc.table_name as bq_table,
    jvds.deleted as deleted,
    datasets.id as dataset_id,
    jv.id as joinable_view_id,
    dp.group,
    jv.name as joinable_view_name,
    jv.created as joinable_view_created,
    jv.expires as joinable_view_expires
FROM
    (
        (
            joinable_views jv
            INNER JOIN joinable_views_datasource jvds ON jv.id = jvds.joinable_view_id
        )
        INNER JOIN (
            (
                datasource_bigquery dsrc
                LEFT JOIN datasets ON dsrc.dataset_id = datasets.id
            )
        ) ON jvds.datasource_id = dsrc.id
    )
    LEFT JOIN dataproducts dp ON datasets.dataproduct_id = dp.id
WHERE
    jv.id = @id;

-- name: GetJoinableViewsToBeDeletedWithRefDatasource :many
SELECT
    jv.id as joinable_view_id,
    jv.name as joinable_view_name,
    bq.project_id as bq_project_id,
    bq.dataset as bq_dataset_id,
    bq.table_name as bq_table_id
FROM
    joinable_views jv
    JOIN joinable_views_datasource jvds ON jv.id = jvds.joinable_view_id
    JOIN datasource_bigquery bq ON bq.id = jvds.datasource_id
WHERE
    jvds.deleted IS NOT NULL;