-- +goose Up

CREATE TABLE access_documentation (
    "access_id"  uuid NOT NULL,
    "polly_id"   TEXT NOT NULL,
    "polly_name" TEXT NOT NULL,
    "polly_url"  TEXT NOT NULL,
    CONSTRAINT fk_dataproduct_access_documentation
        FOREIGN KEY (access_id)
            REFERENCES dataproduct_access (id) ON DELETE CASCADE
);
