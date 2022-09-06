-- +goose Up
ALTER TABLE dataproducts ADD COLUMN "product_area_id" TEXT;

-- +goose Down
ALTER TABLE dataproducts DROP COLUMN "product_area_id";