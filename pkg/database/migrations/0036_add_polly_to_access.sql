-- +goose Up
ALTER TABLE dataproduct_access ADD COLUMN polly_id TEXT;
