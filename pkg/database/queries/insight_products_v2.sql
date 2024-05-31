-- name: GetInsightProductsByProductArea :many
SELECT
    *
FROM
    insight_product_with_teamkatalogen_view
WHERE
    team_id = ANY(@team_id::text[])
ORDER BY
    last_modified DESC;

-- name: GetInsightProductsByGroups :many
SELECT
    *
FROM
    insight_product_with_teamkatalogen_view
WHERE
    "group" = ANY(@groups::text[])
ORDER BY
    last_modified DESC;

-- name: GetInsightProductWithTeamkatalogen :one
SELECT
    *
FROM
    insight_product_with_teamkatalogen_view
WHERE
    "id" = @id
ORDER BY
    last_modified DESC;