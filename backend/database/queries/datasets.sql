-- name: GetDatasets :many
SELECT * FROM datasets WHERE dataproduct_id = @dataproduct_id;

-- name: DeleteDataset :exec
DELETE FROM datasets WHERE id = @id;

-- name: CreateDataset :one
INSERT INTO datasets (
	"dataproduct_id",
	"name",
	"description",
	"pii"
) VALUES (
	@dataproduct_id,
	@name,
	@description,
	@pii
) RETURNING *;

-- name: UpdateDataset :one
UPDATE datasets SET
	"dataproduct_id" = @dataproduct_id,
	"name" = @name,
	"description" = @description,
	"pii" = @pii
WHERE id = @id
RETURNING *;
