-- name: GetDataproduct :one
SELECT *
FROM dataproducts
WHERE id = @id;

-- name: GetDataproducts :many
SELECT *
FROM dataproducts
ORDER BY last_modified DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetDataproductsByIDs :many
SELECT *
FROM dataproducts
WHERE id = ANY (@ids::uuid[])
ORDER BY last_modified DESC;

-- name: GetDataproductsByGroups :many
SELECT *
FROM dataproducts
WHERE "group" = ANY (@groups::text[])
ORDER BY last_modified DESC;

-- name: DeleteDataproduct :exec
DELETE
FROM dataproducts
WHERE id = @id;

-- name: CreateDataproduct :one
INSERT INTO dataproducts ("name",
                          "description",
                          "group",
                          "teamkatalogen_url",
                          "slug",
                          "team_contact",
                          "product_area_id")
VALUES (@name,
        @description,
        @owner_group,
        @owner_teamkatalogen_url,
        @slug,
        @team_contact,
        @product_area_id)
RETURNING *;

-- name: UpdateDataproduct :one
UPDATE dataproducts
SET "name"              = @name,
    "description"       = @description,
    "slug"              = @slug,
    "teamkatalogen_url" = @owner_teamkatalogen_url,
    "team_contact"      = @team_contact,
    "product_area_id"   = @product_area_id
WHERE id = @id
RETURNING *;


-- name: DataproductKeywords :many
SELECT keyword::text, count(1) as "count"
FROM (
	SELECT unnest(ds.keywords) as keyword
	FROM dataproducts dp
    INNER JOIN datasets ds ON ds.dataproduct_id = dp.id
) s
WHERE true
AND CASE WHEN coalesce(TRIM(@keyword), '') = '' THEN true ELSE keyword ILIKE @keyword::text || '%' END
GROUP BY keyword
ORDER BY "count" DESC
LIMIT 15;

-- name: DataproductGroupStats :many
SELECT "group",
       count(1) as "count"
FROM "dataproducts"
GROUP BY "group"
ORDER BY "count" DESC
LIMIT @lim OFFSET @offs;
