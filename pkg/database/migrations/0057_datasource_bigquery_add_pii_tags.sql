-- +goose Up
ALTER TABLE datasource_bigquery
    ADD COLUMN "pii_tags" JSONB;

-- +goose Down
ALTER TABLE datasource_bigquery 
    DROP COLUMN "pii_tags";