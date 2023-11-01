-- +goose Up
CREATE TYPE datasource_type AS ENUM ('bigquery');

CREATE TABLE dataproducts
(
    "id"            uuid                 DEFAULT uuid_generate_v4(),
    "name"          TEXT        NOT NULL,
    "description"   TEXT,
    "group"         TEXT        NOT NULL,
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
    "teamkatalogen_url" TEXT,
    PRIMARY KEY (id)
);

CREATE INDEX dataproducts_tsv_document_idx ON dataproducts USING GIN (tsv_document);

CREATE TRIGGER dataproducts_set_modified
    BEFORE UPDATE
    ON dataproducts
    FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

CREATE TABLE datasource_bigquery
(
    "dataproduct_id" uuid  NOT NULL,
    "project_id"     TEXT  NOT NULL,
    "dataset"        TEXT  NOT NULL,
    "table_name"     TEXT  NOT NULL,
    "schema"         JSONB,
    "last_modified"  TIMESTAMPTZ NOT NULL,
    "created"        TIMESTAMPTZ NOT NULL,
    "expires"        TIMESTAMPTZ,
    "table_type"     TEXT NOT NULL,
    PRIMARY KEY (dataproduct_id),
    CONSTRAINT fk_bigquery_dataproduct
        FOREIGN KEY (dataproduct_id)
            REFERENCES dataproducts (id) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION update_dataproduct_modified_timestamp() RETURNS TRIGGER AS
$$ BEGIN UPDATE dataproducts SET last_modified = now() WHERE id = NEW.dataproduct_id; RETURN NEW; END; $$
LANGUAGE plpgsql;

CREATE TRIGGER datasource_bigquery_set_modified
    BEFORE UPDATE
    ON datasource_bigquery
    FOR EACH ROW
EXECUTE PROCEDURE update_dataproduct_modified_timestamp();

-- +goose Down
DROP TRIGGER datasource_bigquery_set_modified ON datasource_bigquery;
DROP FUNCTION update_dataproduct_modified_timestamp;
DROP TABLE datasource_bigquery;
DROP TRIGGER dataproducts_set_modified ON dataproducts;
DROP INDEX dataproducts_tsv_document_idx;
DROP TABLE dataproducts;
DROP TYPE datasource_type;
