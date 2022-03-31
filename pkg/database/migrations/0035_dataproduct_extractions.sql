-- +goose Up
CREATE TABLE dataproduct_extractions (
    "id"             uuid                  DEFAULT uuid_generate_v4(),
    "dataproduct_id" uuid        NOT NULL,
    "email"          TEXT        NOT NULL,
    "object"         TEXT        NOT NULL,
    "job_id"         TEXT        NOT NULL,
    "created"        TIMESTAMPTZ NOT NULL  DEFAULT NOW(),
    "ready"          BOOLEAN     NOT NULL  DEFAULT false,
    "expired"        BOOLEAN     NOT NULL  DEFAULT false,
    PRIMARY KEY (id)
);
