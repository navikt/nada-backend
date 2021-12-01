-- +goose Up
CREATE TABLE third_party_mappings(
    "dataproduct_id" uuid NOT NULL,
    "services" TEXT[] NOT NULL,
    CONSTRAINT fk_tpm_bigquery_dataproduct
    FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE);
