-- name: UpsertProductArea :exec
INSERT INTO
    tk_product_areas (id, name)
VALUES
    ($1, $2) ON CONFLICT (id) DO
UPDATE
SET
    name = $2;

-- name: UpsertTeam :exec
INSERT INTO
    tk_teams(id, product_area_id, name)
VALUES
    ($1, $2, $3) ON CONFLICT (id) DO
UPDATE
SET
    product_area_id = $2,
    name = $3;

-- name: GetProductAreas :many
SELECT *
FROM tk_product_areas;

-- name: GetAllTeams :many
SELECT *
FROM tk_teams;

-- name: GetProductArea :one
SELECT *
FROM tk_product_areas
WHERE id = $1;

-- name: GetTeamsInProductArea :many
SELECT *
FROM tk_teams
WHERE product_area_id = $1;