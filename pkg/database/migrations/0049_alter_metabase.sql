-- +goose Up
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

-- +goose Down
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