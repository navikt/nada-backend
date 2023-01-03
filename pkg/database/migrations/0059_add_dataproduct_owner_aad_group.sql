-- +goose Up
ALTER TABLE dataproducts ADD COLUMN "aad_group" TEXT;

-- +goose Down
ALTER TABLE dataproducts DROP COLUMN "aad_group";