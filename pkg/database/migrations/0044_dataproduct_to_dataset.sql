-- +goose Up
INSERT INTO datasets ("name", pii, "type", slug, repo, keywords, dataproduct_id)
(SELECT "name", pii, "type", slug, repo, keywords, id FROM dataproducts);

-- +goose Down
TRUNCATE TABLE datasets;