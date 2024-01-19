-- name: GetDataproductComplete :many
SELECT *
FROM dataproduct_complete_view
WHERE dataproduct_id = @id;