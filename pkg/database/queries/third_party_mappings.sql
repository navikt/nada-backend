-- name: MapDataproduct :exec
INSERT INTO third_party_mappings (
    "dataproduct_id",
    "services"
) VALUES (
    @dataproduct_id,
    @services
) ON CONFLICT ("dataproduct_id") DO UPDATE SET
    "services" = EXCLUDED.services;

-- name: GetDataproductMappings :one
SELECT *
FROM third_party_mappings
WHERE "dataproduct_id" = @dataproduct_id;

-- name: GetDataproductsByMapping :many
SELECT dataproducts.* FROM third_party_mappings
INNER JOIN dataproducts ON dataproducts.id = third_party_mappings.dataproduct_id
WHERE @service::TEXT = ANY("services")
LIMIT @lim OFFSET @offs;
