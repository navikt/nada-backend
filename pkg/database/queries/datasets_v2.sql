-- name: GetDatasetComplete :many
SELECT
  *
FROM
  dataset_view
WHERE
  ds_id = @id;