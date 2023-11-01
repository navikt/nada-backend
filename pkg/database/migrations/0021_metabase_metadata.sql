-- +goose Up
CREATE TABLE metabase_metadata (
    "dataproduct_id" uuid NOT NULL,
    "database_id" INT NOT NULL,
    "permission_group_id" INT,
    "sa_email" TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (dataproduct_id),
    CONSTRAINT fk_metabase_metadata
    FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE);

-- +goose Down
DROP TABLE metabase_metadata;
