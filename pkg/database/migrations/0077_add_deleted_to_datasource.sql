-- +goose Up
ALTER TABLE datasource_bigquery ADD COLUMN "deleted" TIMESTAMPTZ;

-- +goose Down
ALTER TABLE datasource_bigquery DROP COLUMN "deleted";