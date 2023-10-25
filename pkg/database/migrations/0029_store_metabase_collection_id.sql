-- +goose Up
ALTER TABLE metabase_metadata ADD COLUMN "collection_id" INT;

-- +goose Down
ALTER TABLE metabase_metadata DROP COLUMN "collection_id";
