-- +goose Up
ALTER TABLE metabase_metadata ADD COLUMN deleted_at TIMESTAMPTZ;