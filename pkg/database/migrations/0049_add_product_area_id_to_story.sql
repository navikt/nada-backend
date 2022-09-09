-- +goose Up
ALTER TABLE stories ADD COLUMN "product_area_id" TEXT;

-- +goose Down
ALTER TABLE stories DROP COLUMN "product_area_id";