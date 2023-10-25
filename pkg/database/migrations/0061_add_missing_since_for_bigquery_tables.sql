-- +goose Up
ALTER TABLE datasource_bigquery ADD COLUMN "missing_since" TIMESTAMPTZ;

-- +goose Down
ALTER TABLE datasource_bigquery DROP COLUMN "missing_since";