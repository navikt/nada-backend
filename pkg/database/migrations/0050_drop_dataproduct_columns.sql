-- +goose Up
ALTER TABLE dataproducts DROP COLUMN pii, DROP COLUMN "type";

-- +goose Down
ALTER TABLE dataproducts 
    ADD COLUMN pii BOOLEAN NOT NULL DEFAULT TRUE, 
    ADD COLUMN "type" datasource_type NOT NULL DEFAULT 'bigquery';