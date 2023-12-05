-- +goose Up
ALTER TABLE dataproducts DROP COLUMN product_area_id;
ALTER TABLE quarto_stories DROP COLUMN product_area_id;
ALTER TABLE stories DROP COLUMN product_area_id;
ALTER TABLE insight_product DROP COLUMN product_area_id;

-- +goose Down
ALTER TABLE dataproducts ADD COLUMN product_area_id TEXT;
ALTER TABLE quarto_stories ADD COLUMN product_area_id TEXT;
ALTER TABLE stories ADD COLUMN product_area_id TEXT;
ALTER TABLE insight_product ADD COLUMN product_area_id TEXT;
