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
    id = ANY (@id :: uuid [])
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


-- name: GetInsightProductsNumberByTeam :one
SELECT
    COUNT(*) as "count"
FROM
    insight_product
WHERE
    team_id = @team_id;

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
    "team_id" = @team_id
WHERE
    id = @id RETURNING *;

-- name: DeleteInsightProduct :exec
DELETE FROM
    insight_product
WHERE
    id = @id;

-- name: GetInsightProductByGroups_ :many
SELECT
    *
FROM
    insight_product
WHERE
    "group" = ANY (@groups :: text [])
ORDER BY
    last_modified DESC;
