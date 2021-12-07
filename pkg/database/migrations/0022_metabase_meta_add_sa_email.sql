-- +goose Up
ALTER TABLE metabase_metadata ADD COLUMN sa_email TEXT NOT NULL DEFAULT '';
