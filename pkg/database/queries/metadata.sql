-- name: GetDatasetMetadata :one
SELECT * FROM dataset_metadata WHERE dataset_id = @dataset_id;

-- name: WriteDatasetMetadata :exec
INSERT INTO dataset_metadata (
	"dataset_id",
	"schema"
) VALUES (
	@dataset_id,
	@schema
)
ON CONFLICT (dataset_id) DO UPDATE
SET
    "dataset_id" = @dataset_id,
    "schema" = @schema;
