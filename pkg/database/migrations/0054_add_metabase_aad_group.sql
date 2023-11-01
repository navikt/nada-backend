-- +goose Up
ALTER TABLE metabase_metadata ADD COLUMN "aad_premission_group_id" INT;

-- +goose Down
ALTER TABLE metabase_metadata DROP COLUMN "aad_premission_group_id";