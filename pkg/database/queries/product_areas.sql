-- name: UpsertProductArea :one
INSERT INTO "product_areas" (
    "external_id",
    "name"
) VALUES (
    @external_id,
    @name
) ON CONFLICT (external_id)
DO 
UPDATE SET "name" = EXCLUDED.name
RETURNING *;

-- name: GetAllProductAreas :many
SELECT * FROM "product_areas";

-- name: GetProductAreaForExternalID :one
SELECT * FROM "product_areas"
WHERE external_id = @external_id;

-- name: GetProductArea :one
SELECT * FROM "product_areas"
WHERE id = @id;
