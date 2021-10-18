-- name: GetCollection :one
SELECT * FROM collections WHERE id = @id;

-- name: GetCollections :many
SELECT * FROM collections ORDER BY last_modified DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = @id;

-- name: CreateCollection :one
INSERT INTO collections (
	"name",
	"description",
	"slug",
	"repo",
	"group",
	"keywords"
) VALUES (
	@name,
	@description,
	@slug,
	@repo,
	@owner_group,
	@keywords
) RETURNING *;

-- name: UpdateCollection :one
UPDATE collections SET
	"name" = @name,
	"description" = @description,
	"slug" = @slug,
	"repo" = @repo,
	"keywords" = @keywords
WHERE id = @id
RETURNING *;

-- name: CreateCollectionElement :exec
INSERT INTO collection_elements (
	"element_id",
	"collection_id",
	"element_type"
) VALUES (
	@element_id,
	@collection_id,
	@element_type
);

-- name: DeleteCollectionElement :exec
DELETE FROM collection_elements WHERE element_id = @element_id AND collection_id = @collection_id AND element_type = @element_type;

-- name: GetCollectionElements :many
SELECT * FROM collection_elements WHERE collection_id = @collection_id;
