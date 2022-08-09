-- +goose Up
ALTER TABLE dataset_requesters 
    DROP CONSTRAINT fk_requester_dataproduct;
ALTER TABLE dataset_requesters
    ADD COLUMN dataset_id uuid;
UPDATE dataset_requesters a SET dataset_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);
ALTER TABLE dataset_requesters
    ALTER COLUMN dataset_id SET NOT NULL;
ALTER TABLE dataset_requesters DROP CONSTRAINT dataproduct_requesters_pkey;
ALTER TABLE dataset_requesters ADD PRIMARY KEY (dataset_id, "subject");

ALTER TABLE dataset_requesters
    DROP COLUMN dataproduct_id;
ALTER TABLE dataset_requesters
    ADD CONSTRAINT fk_requester_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;    

-- +goose Down
ALTER TABLE dataset_requesters
    DROP CONSTRAINT fk_requester_dataset;
ALTER TABLE dataset_requesters
    ADD COLUMN dataproduct_id uuid;
UPDATE dataset_requesters a SET dataproduct_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = a.dataset_id);
ALTER TABLE dataset_requesters
    ALTER COLUMN dataproduct_id SET NOT NULL;
ALTER TABLE dataset_requesters DROP CONSTRAINT dataset_requesters_pkey;
ALTER TABLE dataset_requesters ADD PRIMARY KEY (dataproduct_id, "subject");

ALTER TABLE dataset_requesters
    DROP COLUMN dataset_id;
ALTER TABLE dataset_requesters
    ADD CONSTRAINT fk_requester_dataproduct FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;