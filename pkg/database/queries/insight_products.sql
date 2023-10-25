-- name: CreateInsightProduct :one
INSERT INTO
    insight_product (
        "name",
        "creator",
        "description",
        "type",
        "link",
        "keywords",
        "group",
        "teamkatalogen_url",
        "product_area_id",
        "team_id"
    )
VALUES
    (
        @name,
        @creator,
        @description,
        @type,
        @link,
        @keywords,
        @owner_group,
        @teamkatalogen_url,
        @product_area_id,
        @team_id
    ) RETURNING *;

-- name: GetInsightProduct :one
SELECT
    *
FROM
    insight_product
WHERE
    id = @id;

-- name: GetInsightProducts :many
SELECT
    *
FROM
    insight_product
ORDER BY
    last_modified DESC;

-- name: GetInsightProductsByIDs :many
SELECT
    *
FROM
    insight_product
WHERE
    id = ANY (@ids :: uuid [])
ORDER BY
    last_modified DESC;

-- name: GetInsightProductsByProductArea :many
SELECT
    *
FROM
    insight_product
WHERE
    product_area_id = @product_area_id
ORDER BY
    last_modified DESC;

-- name: GetInsightProductsByTeam :many
SELECT
    *
FROM
    insight_product
WHERE
    team_id = @team_id
ORDER BY
    last_modified DESC;

-- name: UpdateInsightProduct :one
UPDATE
    insight_product
SET
    "name" = @name,
    "creator" = @creator,
    "description" = @description,
    "type" = @type,
    "link" = @link,
    "keywords" = @keywords,
    "teamkatalogen_url" = @teamkatalogen_url,
    "product_area_id" = @product_area_id,
    "team_id" = @team_id
WHERE
    id = @id RETURNING *;

-- name: DeleteInsightProduct :exec
DELETE FROM
    insight_product
WHERE
    id = @id;

-- name: GetInsightProductByGroups :many
SELECT
    *
FROM
    insight_product
WHERE
    "group" = ANY (@groups :: text [])
ORDER BY
    last_modified DESC;