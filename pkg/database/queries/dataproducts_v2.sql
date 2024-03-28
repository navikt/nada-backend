-- name: GetDataproductsWithDatasets :many
SELECT dp.*, dsrc.last_modified as "dsrc_last_modified"
FROM dataproduct_view dp
LEFT JOIN datasource_bigquery dsrc ON dsrc.dataset_id = dp.ds_id
WHERE (array_length(@ids::uuid[], 1) IS NULL OR dp_id = ANY (@ids))
 AND (array_length(@groups::TEXT[], 1) IS NULL OR dp_group = ANY (@groups));


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