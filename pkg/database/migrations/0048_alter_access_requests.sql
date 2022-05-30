-- +goose Up
ALTER TABLE dataset_access_requests 
    DROP CONSTRAINT fk_requester_dataproduct;
UPDATE dataset_access_requests SET dataproduct_id = (SELECT id FROM datasets WHERE dataproduct_id = dataproduct_id);

ALTER TABLE dataset_access_requests
    RENAME COLUMN dataproduct_id TO dataset_id;
ALTER TABLE dataset_access_requests
    ADD CONSTRAINT fk_requester_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;
    

-- +goose Down
ALTER TABLE dataset_access_requests
    DROP CONSTRAINT fk_requester_dataset;
UPDATE dataset_access_requests SET dataset_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = dataset_id);

ALTER TABLE dataset_access_requests
    RENAME COLUMN dataset_id TO dataproduct_id;

ALTER TABLE dataset_access_requests
    ADD CONSTRAINT fk_requester_dataproduct FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;