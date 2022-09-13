-- +goose Up
ALTER TABLE dataproducts ADD COLUMN "team_id" TEXT;
ALTER TABLE stories ADD COLUMN "team_id" TEXT;

-- +goose Down
ALTER TABLE dataproducts DROP COLUMN "team_id" TEXT;
ALTER TABLE stories DROP COLUMN "team_id" TEXT;
