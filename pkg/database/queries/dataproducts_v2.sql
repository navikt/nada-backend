-- name: GetDataproductWithDatasets :many
SELECT *
FROM dataproduct_view
WHERE dp_id = @id;

-- name: GetDataproductWithDatasetsBasic :many
SELECT *
FROM dataproduct_with_teamkatalogen_view dp LEFT JOIN datasets ds ON ds.dataproduct_id = dp.id
WHERE dp.id = @id;

-- name: GetDataproductKeywords :many
SELECT DISTINCT unnest(keywords)::text FROM datasets ds WHERE ds.dataproduct_id = @dpid;

-- name: GetDataproductsNumberByTeam :one
SELECT COUNT(*) as "count"
FROM dataproducts
WHERE team_id = @team_id;