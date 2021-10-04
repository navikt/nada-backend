-- name: SearchDataproducts :many
SELECT * FROM "dataproducts" WHERE "tsv_document" @@ websearch_to_tsquery('norwegian', @query);

-- name: SearchDatasets :many
SELECT * FROM "datasets" WHERE "tsv_document" @@ websearch_to_tsquery('norwegian', @query);
