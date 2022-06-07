-- +goose Up
ALTER TABLE third_party_mappings 
    DROP CONSTRAINT fk_tpm_bigquery_dataproduct;
ALTER TABLE third_party_mappings
    ADD COLUMN dataset_id uuid;
UPDATE third_party_mappings a SET dataset_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);
ALTER TABLE third_party_mappings
    ALTER COLUMN dataset_id SET NOT NULL;
ALTER TABLE third_party_mappings DROP CONSTRAINT third_party_mappings_pkey;
ALTER TABLE third_party_mappings ADD PRIMARY KEY (dataset_id);

ALTER TABLE third_party_mappings
    DROP COLUMN dataproduct_id;
ALTER TABLE third_party_mappings
    ADD CONSTRAINT fk_tpm_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;    

ALTER TABLE dataset_access 
    DROP CONSTRAINT fk_access_dataproduct;
UPDATE dataset_access a SET dataproduct_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);
ALTER TABLE dataset_access
    RENAME COLUMN dataproduct_id TO dataset_id;
ALTER TABLE dataset_access
    ADD CONSTRAINT fk_access_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE third_party_mappings
    DROP CONSTRAINT fk_tpm_dataset;
ALTER TABLE third_party_mappings
    ADD COLUMN dataproduct_id uuid;
UPDATE third_party_mappings SET dataproduct_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = dataset_id);
ALTER TABLE third_party_mappings
    ALTER COLUMN dataproduct_id SET NOT NULL;
ALTER TABLE third_party_mappings DROP CONSTRAINT third_party_mappings_pkey;
ALTER TABLE third_party_mappings ADD PRIMARY KEY (dataproduct_id);

ALTER TABLE third_party_mappings
    DROP COLUMN dataset_id;
ALTER TABLE third_party_mappings
    ADD CONSTRAINT fk_tpm_bigquery_dataproduct FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;

ALTER TABLE dataset_access 
    DROP CONSTRAINT fk_access_dataset;
UPDATE dataset_access SET dataset_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = dataset_id);
ALTER TABLE dataset_access
    RENAME COLUMN dataset_id TO dataproduct_id;
ALTER TABLE dataset_access
    ADD CONSTRAINT fk_access_dataproduct FOREIGN KEY (dataproduct_id)
            REFERENCES dataproducts (id) ON DELETE CASCADE;