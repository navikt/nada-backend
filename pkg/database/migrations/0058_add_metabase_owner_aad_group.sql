-- +goose Up
ALTER TABLE metabase_metadata ADD COLUMN "aad_owner_group_id" INT;

-- +goose Down
ALTER TABLE metabase_metadata DROP COLUMN "aad_owner_group_id";