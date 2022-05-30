-- +goose Up
ALTER TABLE datasource_bigquery 
    DROP CONSTRAINT fk_bigquery_dataproduct;
UPDATE datasource_bigquery SET dataproduct_id = (SELECT id FROM datasets WHERE dataproduct_id = dataproduct_id);

ALTER TABLE datasource_bigquery
    RENAME COLUMN dataproduct_id TO dataset_id;
ALTER TABLE datasource_bigquery
    ADD CONSTRAINT fk_bigquery_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;
    

-- +goose Down
ALTER TABLE datasource_bigquery
    DROP CONSTRAINT fk_bigquery_dataset;
UPDATE datasource_bigquery SET dataset_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = dataset_id);

ALTER TABLE datasource_bigquery
    RENAME COLUMN dataset_id TO dataproduct_id;

ALTER TABLE datasource_bigquery
    ADD CONSTRAINT fk_bigquery_dataproduct FOREIGN_KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;