-- +goose Up
CREATE TABLE dataproduct_extractions (
    "id"             uuid                  DEFAULT uuid_generate_v4(),
    "dataproduct_id" uuid        NOT NULL,
    "email"          TEXT        NOT NULL,
    "bucket_path"    TEXT        NOT NULL,
    "job_id"         TEXT        NOT NULL,
    "created"        TIMESTAMPTZ NOT NULL  DEFAULT NOW(),
    "ready_at"       TIMESTAMPTZ,
    "expired_at"     TIMESTAMPTZ,
    PRIMARY KEY (id)
);
