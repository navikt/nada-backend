-- +goose Up
ALTER TABLE dataproducts ADD COLUMN "team_contact" TEXT;

-- +goose Down
ALTER TABLE dataproducts DROP COLUMN "team_contact";