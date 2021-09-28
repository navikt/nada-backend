-- name: GetDataproducts :many
SELECT * FROM dataproducts;

-- name: DeleteDataproduct :exec
DELETE FROM dataproducts WHERE id = @id;

-- name: CreateDataproduct :one
INSERT INTO dataproducts (
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

-- name: UpdateDataproduct :one
UPDATE dataproducts SET
	"name" = @name,
	"description" = @description,
	"slug" = @slug,
	"repo" = @repo,
	"team" = @team,
	"keywords" = @keywords
WHERE id = @id
RETURNING *;
