-- name: GetDataproductsWithDatasets :many
SELECT dp.*, dsrc.last_modified as "dsrc_last_modified"
FROM dataproduct_view dp
LEFT JOIN datasource_bigquery dsrc ON dsrc.dataset_id = dp.ds_id
WHERE (array_length(@ids::uuid[], 1) IS NULL OR dp_id = ANY (@ids))
 AND (array_length(@groups::TEXT[], 1) IS NULL OR dp_group = ANY (@groups))
ORDER BY ds_name ASC;

-- name: GetDataproductsWithDatasetsAndAccessRequests :many
SELECT dp.*, dsrc.last_modified as "dsrc_last_modified",
 dar.id as "dar_id", dar.dataset_id as "dar_dataset_id", dar.subject as "dar_subject", dar.owner as "dar_owner",
  dar.expires as "dar_expires", dar.status as "dar_status", dar.granter as "dar_granter", dar.reason as "dar_reason", 
  dar.closed as "dar_closed", dar.polly_documentation_id as "dar_polly_documentation_id", dar.created as "dar_created"
FROM dataproduct_view dp
LEFT JOIN datasource_bigquery dsrc ON dsrc.dataset_id = dp.ds_id
LEFT JOIN dataset_access_requests dar ON dar.dataset_id = dp.ds_id AND dar.status = 'pending'
WHERE (array_length(@ids::uuid[], 1) IS NULL OR dp_id = ANY (@ids))
 AND (array_length(@groups::TEXT[], 1) IS NULL OR dp_group = ANY (@groups))
ORDER by dp.dp_group, dp.dp_name;

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