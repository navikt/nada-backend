-- name: SearchDataproducts :many
SELECT * FROM "dataproducts" WHERE "tsv_document" @@ websearch_to_tsquery('norwegian', sqlc.arg('query')) LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: SearchDatasets :many
SELECT * FROM "datasets" WHERE "tsv_document" @@ websearch_to_tsquery('norwegian', sqlc.arg('query')) LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
