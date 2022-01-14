-- +goose Up
ALTER TABLE metabase_metadata ADD COLUMN "collection_id" INT;