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

-- +goose Down
DROP FUNCTION update_dataset_modified_timestamp;
DROP TRIGGER datasets_set_modified ON datasets;
DROP INDEX datasets_tsv_document_idx;
DROP TABLE datasets;
