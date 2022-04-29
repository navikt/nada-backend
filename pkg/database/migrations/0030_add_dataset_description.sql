-- +goose Up
ALTER TABLE datasource_bigquery ADD COLUMN "description" TEXT;

-- +goose Down
ALTER TABLE datasource_bigquery DROP COLUMN description;
