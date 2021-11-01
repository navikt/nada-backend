-- +goose Up

ALTER TABLE datasource_bigquery 
ADD COLUMN last_modified TIMESTAMPTZ NOT NULL,
ADD COLUMN created TIMESTAMPTZ NOT NULL,
ADD COLUMN expires TIMESTAMPTZ,
ADD COLUMN table_type TEXT NOT NULL;
