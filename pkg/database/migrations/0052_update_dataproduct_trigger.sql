-- +goose Up
DROP TRIGGER datasource_bigquery_set_modified ON datasource_bigquery;

CREATE TRIGGER dataproducts_set_modified
    BEFORE UPDATE
    ON datasets
    FOR EACH ROW
EXECUTE PROCEDURE update_dataproduct_modified_timestamp();

CREATE TRIGGER datasets_set_modified
    BEFORE UPDATE
    ON datasource_bigquery
    FOR EACH ROW
EXECUTE PROCEDURE update_dataset_modified_timestamp();

-- +goose Down
DROP TRIGGER dataproducts_set_modified ON datasets;

CREATE TRIGGER datasource_bigquery_set_modified
    BEFORE UPDATE
    ON datasource_bigquery
    FOR EACH ROW
EXECUTE PROCEDURE update_dataproduct_modified_timestamp();
