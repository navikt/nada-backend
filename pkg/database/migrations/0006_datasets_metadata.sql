-- +goose Up
CREATE TABLE dataset_metadata (
    "dataset_id" uuid NOT NULL,
    "created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "schema" JSONB NOT NULL,
    PRIMARY KEY(dataset_id),
		CONSTRAINT fk_dataset
			FOREIGN KEY(dataset_id)
				REFERENCES datasets(id) 
);

CREATE TRIGGER dataset_metadata_set_modified
BEFORE UPDATE ON dataset_metadata
FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

-- +goose Down
DROP TABLE dataset_metadata;
DROP TRIGGER dataset_metadata_set_modified;
