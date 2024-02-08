-- name: GetDataproductWithDatasets :many
SELECT *
FROM dataproduct_view
WHERE dp_id = @id;