-- +goose Up
ALTER TABLE metabase_metadata ADD COLUMN deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE metabase_metadata DROP COLUMN deleted_at;