-- name: CreateQuarto :one
INSERT INTO "quarto" ("owner",
                      "content")
VALUES (
    @team,
    @content
)
RETURNING *;

-- name: UpdateQuarto :one
UPDATE "quarto"
SET content = @content
WHERE id = @id
RETURNING *;

-- name: DeleteQuarto :exec
DELETE FROM "quarto"
WHERE id = @id;

-- name: GetQuarto :one
SELECT *
FROM "quarto"
WHERE id = @id;

-- name: GetQuartos :many
SELECT *
FROM "quarto";

-- name: GetQuartosForOwner :many
SELECT *
FROM "quarto"
WHERE owner = @owner;
