-- name: GetDataproductCollection :one
SELECT * FROM dataproduct_collections WHERE id = @id;

-- name: GetDataproductCollections :many
SELECT * FROM dataproduct_collections ORDER BY last_modified DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DeleteDataproductCollection :exec
DELETE FROM dataproduct_collections WHERE id = @id;

-- name: CreateDataproductCollection :one
INSERT INTO dataproduct_collections (
	"name",
	"description",
	"slug",
	"repo",
	"team",
	"keywords"
) VALUES (
	@name,
	@description,
	@slug,
	@repo,
	@team,
	@keywords
) RETURNING *;

-- name: UpdateDataproductCollection :one
UPDATE dataproduct_collections SET
	"name" = @name,
	"description" = @description,
	"slug" = @slug,
	"repo" = @repo,
	"team" = (SELECT team FROM dataproduct_collections dp WHERE dp.id = @id),
	"keywords" = @keywords
WHERE id = @id
RETURNING *;
