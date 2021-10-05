-- name: GetDataset :one
SELECT * FROM datasets WHERE id = @id;

-- name: GetDatasets :many
SELECT * FROM datasets ORDER BY last_modified DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetDatasetsForDataproduct :many
SELECT id, name, type FROM datasets WHERE dataproduct_id = @dataproduct_id;

-- name: DeleteDataset :exec
DELETE FROM datasets WHERE id = @id;

-- name: CreateDataset :one
INSERT INTO datasets (
	"dataproduct_id",
	"name",
	"description",
	"pii",
	"project_id",
	"dataset",
	"table_name",
	"type"
) VALUES (
	@dataproduct_id,
	@name,
	@description,
	@pii,
	@project_id,
	@dataset,
	@table_name,
	@type
) RETURNING *;

-- name: UpdateDataset :one
UPDATE datasets SET
	"dataproduct_id" = @dataproduct_id,
	"name" = @name,
	"description" = @description,
	"pii" = @pii
WHERE id = @id
RETURNING *;
