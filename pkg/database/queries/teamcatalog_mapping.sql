-- name: GetTeamsAndProductAreaIDs :many
SELECT *
FROM "team_productarea_mapping";

-- name: GetTeamAndProductAreaID :one
SELECT *
FROM "team_productarea_mapping"
WHERE team_id = @team_id;

-- name: CreateTeamAndProductAreaMapping :one
INSERT INTO "team_productarea_mapping" (
    "team_id",
    "product_area_id"
) VALUES (
    @team_id,
    @product_area_id
) RETURNING *;

-- name: UpdateProductAreaForTeam :exec
UPDATE "team_productarea_mapping"
SET product_area_id = @product_area_id
WHERE team_id = @team_id;
