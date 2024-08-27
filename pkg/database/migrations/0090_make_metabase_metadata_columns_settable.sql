-- +goose Up
ALTER TABLE metabase_metadata
    ADD COLUMN sync_completed timestamptz DEFAULT NULL;

ALTER TABLE metabase_metadata
    ALTER COLUMN database_id DROP NOT NULL;

UPDATE metabase_metadata
SET sync_completed = NOW();

-- +goose Down
ALTER TABLE metabase_metadata
    DROP COLUMN sync_completed;

DELETE FROM metabase_metadata
WHERE database_id IS NULL;

ALTER TABLE metabase_metadata
    ALTER COLUMN database_id SET NOT NULL;
