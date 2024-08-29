-- name: GetAllDatasetsMinimal :many
SELECT ds.id, ds.created, name, project_id, dataset, table_name 
FROM datasets ds 
JOIN datasource_bigquery dsb 
ON ds.id = dsb.dataset_id;

-- name: GetDatasetComplete :many
SELECT
  *
FROM
  dataset_view
WHERE
  ds_id = @id;

-- name: GetAccessibleDatasets :many
SELECT
  DISTINCT ON (ds.id)
  ds.*,
  dsa.subject AS "subject",
  dsa.owner AS "access_owner",
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
  OR (
    SPLIT_PART(dsa.subject, ':', 1) != 'serviceAccount'
    AND (
        @requester::TEXT IS NOT NULL
        AND dsa.subject = LOWER(@requester)
        OR SPLIT_PART(dsa.subject, ':', 2) = ANY(@groups::TEXT[])
    )
  )
  AND revoked IS NULL
  AND (
    expires > NOW()
    OR expires IS NULL
  )
ORDER BY
  ds.id,
  ds.last_modified DESC;

-- name: GetAccessibleDatasetsByOwnedServiceAccounts :many
SELECT
  ds.*,
  dsa.subject AS "subject",
  dsa.owner AS "access_owner",
  dp.slug AS dp_slug,
  dp.name AS dp_name,
  dp.group
FROM
  datasets ds
  LEFT JOIN dataproducts dp ON ds.dataproduct_id = dp.id
  LEFT JOIN dataset_access dsa ON dsa.dataset_id = ds.id
WHERE
  SPLIT_PART("subject", ':', 1) = 'serviceAccount'
  AND (
    dsa.owner = @requester
    OR dsa.owner = ANY(@groups::TEXT[])
  )  
  AND revoked IS NULL
  AND (
    expires > NOW()
    OR expires IS NULL
  )
ORDER BY
  ds.last_modified DESC;