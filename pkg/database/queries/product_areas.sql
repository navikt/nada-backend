-- name: AddProductArea :one
INSERT INTO "product_areas" (
    "external_id",
    "name"
) VALUES (
    @external_id,
    @name
)
RETURNING *;

-- name: GetAllProductAreas :many
SELECT * FROM "product_areas";

-- name: GetProductArea :one
SELECT * FROM "product_areas"
WHERE id = @id;
