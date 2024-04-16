-- name: GetAllDatasets :many
SELECT
  *
FROM 
  dataset_view;

-- name: GetDatasetComplete :many
SELECT
  *
FROM
  dataset_view
WHERE
  ds_id = @id;

-- name: GetAccessibleDatasets :many
SELECT
  ds.*,
  dp.slug AS dp_slug,
  dp.name AS dp_name,
  dp.group
FROM
  datasets ds
  LEFT JOIN dataproducts dp ON ds.dataproduct_id = dp.id
  LEFT JOIN dataset_access dsa ON dsa.dataset_id = ds.id
WHERE
  array_length(@groups::TEXT[], 1) IS NOT NULL AND array_length(@groups::TEXT[], 1)!=0
  AND dp.group = ANY(@groups :: TEXT [])
  OR @requester::TEXT IS NOT NULL
  AND dsa.subject = LOWER(@requester)
  AND revoked IS NULL
  AND (
    expires > NOW()
    OR expires IS NULL
  )
ORDER BY
  ds.last_modified DESC;