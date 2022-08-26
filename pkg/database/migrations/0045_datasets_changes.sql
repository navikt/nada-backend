-- +goose Up
ALTER TABLE dataset_access_requests 
    DROP CONSTRAINT fk_requester_dataproduct;
UPDATE dataset_access_requests a SET dataproduct_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);

ALTER TABLE dataset_access_requests
    RENAME COLUMN dataproduct_id TO dataset_id;
ALTER TABLE dataset_access_requests
    ADD CONSTRAINT fk_requester_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;
    
ALTER TABLE metabase_metadata 
    DROP CONSTRAINT fk_metabase_metadata;
ALTER TABLE metabase_metadata
    ADD COLUMN dataset_id uuid;
UPDATE metabase_metadata a SET dataset_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);
ALTER TABLE metabase_metadata
    ALTER COLUMN dataset_id SET NOT NULL;
ALTER TABLE metabase_metadata DROP CONSTRAINT metabase_metadata_pkey;
ALTER TABLE metabase_metadata ADD PRIMARY KEY (dataset_id);
ALTER TABLE metabase_metadata
    DROP COLUMN dataproduct_id;
ALTER TABLE metabase_metadata
    ADD CONSTRAINT fk_metabase_metadata FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;    

ALTER TABLE dataproducts DROP COLUMN pii, DROP COLUMN "type";
ALTER TABLE dataproducts DROP COLUMN repo, DROP COLUMN keywords;

DROP TRIGGER datasource_bigquery_set_modified ON datasource_bigquery;

CREATE TRIGGER dataproducts_datasets_set_modified
    BEFORE UPDATE
    ON datasets
    FOR EACH ROW
EXECUTE PROCEDURE update_dataproduct_modified_timestamp();

CREATE TRIGGER datasets_datasource_bigquery_set_modified
    BEFORE UPDATE
    ON datasource_bigquery
    FOR EACH ROW
EXECUTE PROCEDURE update_dataset_modified_timestamp();

-- +goose Down
DROP TRIGGER dataproducts_datasets_set_modified ON datasets;
DROP TRIGGER datasets_datasource_bigquery_set_modified ON datasource_bigquery;

CREATE TRIGGER datasource_bigquery_set_modified
    BEFORE UPDATE
    ON datasource_bigquery
    FOR EACH ROW
EXECUTE PROCEDURE update_dataproduct_modified_timestamp();


ALTER TABLE dataproducts ADD COLUMN repo TEXT, ADD COLUMN keywords TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE dataproducts 
    ADD COLUMN pii BOOLEAN NOT NULL DEFAULT TRUE, 
    ADD COLUMN "type" datasource_type NOT NULL DEFAULT 'bigquery';

ALTER TABLE metabase_metadata
    DROP CONSTRAINT fk_metabase_metadata;
ALTER TABLE metabase_metadata
    ADD COLUMN dataproduct_id uuid;
UPDATE metabase_metadata a SET dataproduct_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = a.dataset_id);
ALTER TABLE metabase_metadata
    ALTER COLUMN dataproduct_id SET NOT NULL;
ALTER TABLE metabase_metadata DROP CONSTRAINT metabase_metadata_pkey;
ALTER TABLE metabase_metadata ADD PRIMARY KEY (dataproduct_id);

ALTER TABLE metabase_metadata
    DROP COLUMN dataset_id;
ALTER TABLE metabase_metadata
    ADD CONSTRAINT fk_metabase_metadata FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;

ALTER TABLE dataset_access_requests
    DROP CONSTRAINT fk_requester_dataset;
UPDATE dataset_access_requests a SET dataset_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = a.dataset_id);

ALTER TABLE dataset_access_requests
    RENAME COLUMN dataset_id TO dataproduct_id;

ALTER TABLE dataset_access_requests
    ADD CONSTRAINT fk_requester_dataproduct FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;