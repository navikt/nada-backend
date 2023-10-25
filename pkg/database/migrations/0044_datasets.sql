-- +goose Up
CREATE TABLE datasets
(
    "id"            uuid                 DEFAULT uuid_generate_v4(),
    "name"          TEXT        NOT NULL,
    "description"   TEXT,
    "pii"           BOOLEAN     NOT NULL,
    "created"       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "type"          datasource_type  NOT NULL,
    "tsv_document"  tsvector GENERATED ALWAYS AS (
                                to_tsvector('norwegian', "name")
                                || to_tsvector('norwegian', coalesce("description", ''))
                        ) STORED,
    "slug"           TEXT NOT NULL,
    "repo"           TEXT,
    "keywords"       TEXT[] NOT NULL DEFAULT '{}',
    "dataproduct_id" uuid NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_dataset_dataproduct
        FOREIGN KEY (dataproduct_id)
            REFERENCES dataproducts (id) ON DELETE CASCADE
);

CREATE INDEX datasets_tsv_document_idx ON datasets USING GIN (tsv_document);

CREATE TRIGGER datasets_set_modified
    BEFORE UPDATE
    ON datasets
    FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

CREATE OR REPLACE FUNCTION update_dataset_modified_timestamp() RETURNS TRIGGER AS
$$ BEGIN UPDATE datasets SET last_modified = now() WHERE id = NEW.dataset_id; RETURN NEW; END; $$
LANGUAGE plpgsql;

ALTER TABLE dataproduct_requesters RENAME TO dataset_requesters;
ALTER TABLE dataproduct_access RENAME TO dataset_access;
ALTER TABLE dataproduct_access_request RENAME TO dataset_access_requests;

INSERT INTO datasets ("name", pii, "type", slug, repo, keywords, dataproduct_id)
(SELECT "name", pii, "type", slug, repo, keywords, id FROM dataproducts);

ALTER TABLE datasource_bigquery 
    DROP CONSTRAINT fk_bigquery_dataproduct;
UPDATE datasource_bigquery a SET dataproduct_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);

ALTER TABLE datasource_bigquery
    RENAME COLUMN dataproduct_id TO dataset_id;
ALTER TABLE datasource_bigquery
    ADD CONSTRAINT fk_bigquery_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;
    
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
ALTER TABLE dataset_access RENAME TO dataproduct_access;
ALTER TABLE dataset_access_requests RENAME TO dataproduct_access_request;
ALTER TABLE dataset_requesters RENAME TO dataproduct_requesters;

ALTER TABLE dataproduct_requesters
    DROP CONSTRAINT fk_requester_dataset;
ALTER TABLE dataproduct_requesters
    ADD COLUMN dataproduct_id uuid;
UPDATE dataproduct_requesters a SET dataproduct_id = (SELECT dataproduct_id FROM datasets e WHERE e.id = a.dataset_id);
ALTER TABLE dataproduct_requesters
    ALTER COLUMN dataproduct_id SET NOT NULL;
ALTER TABLE dataproduct_requesters DROP CONSTRAINT dataset_requesters_pkey;
ALTER TABLE dataproduct_requesters ADD PRIMARY KEY (dataproduct_id, "subject");

ALTER TABLE dataproduct_requesters
    DROP COLUMN dataset_id;
ALTER TABLE dataproduct_requesters
    ADD CONSTRAINT fk_requester_dataproduct FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;


ALTER TABLE datasource_bigquery
    ADD COLUMN dataproduct_id uuid;
UPDATE datasource_bigquery a SET dataproduct_id = (SELECT dataproduct_id FROM datasets e WHERE e.id = a.dataset_id);
ALTER TABLE datasource_bigquery
    ALTER COLUMN dataproduct_id SET NOT NULL;
ALTER TABLE datasource_bigquery
    DROP CONSTRAINT fk_bigquery_dataset;
ALTER TABLE datasource_bigquery
    DROP COLUMN dataset_id;
ALTER TABLE datasource_bigquery
    ADD CONSTRAINT fk_bigquery_dataproduct FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;

UPDATE dataproducts d SET pii = (SELECT pii FROM datasets e WHERE e.dataproduct_id = d.id);
UPDATE dataproducts d SET repo = (SELECT repo FROM datasets e WHERE e.dataproduct_id = d.id);
UPDATE dataproducts d SET "type" = (SELECT "type" FROM datasets e WHERE e.dataproduct_id = d.id);
UPDATE dataproducts d SET keywords = (SELECT keywords FROM datasets e WHERE e.dataproduct_id = d.id);

TRUNCATE TABLE datasets;

DROP FUNCTION update_dataset_modified_timestamp;
DROP TRIGGER datasets_set_modified ON datasets;
DROP INDEX datasets_tsv_document_idx;
DROP TABLE datasets;
