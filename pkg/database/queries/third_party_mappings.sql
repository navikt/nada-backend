-- name: MapDataset :exec
INSERT INTO third_party_mappings (
    "dataset_id",
    "services"
) VALUES (
    @dataset_id,
    @services
) ON CONFLICT ("dataset_id") DO UPDATE SET
    "services" = EXCLUDED.services;

-- name: GetDatasetMappings :one
SELECT *
FROM third_party_mappings
WHERE "dataset_id" = @dataset_id;

-- name: GetDatasetsByMapping :many
SELECT datasets.* FROM third_party_mappings
INNER JOIN datasets ON datasets.id = third_party_mappings.dataset_id
WHERE @service::TEXT = ANY("services")
LIMIT @lim OFFSET @offs;
