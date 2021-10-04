-- +goose Up

ALTER TABLE datasets
ADD COLUMN "tsv_document" tsvector GENERATED ALWAYS AS (
	   to_tsvector('norwegian', "name")
	|| to_tsvector('norwegian', coalesce("description", ''))
	|| to_tsvector('norwegian', coalesce("project_id", ''))
	|| to_tsvector('norwegian', coalesce("dataset", ''))
	|| to_tsvector('norwegian', coalesce("table_name", ''))
	|| to_tsvector('norwegian', coalesce("type", ''))
) STORED;

CREATE INDEX datasets_tsv_document_idx ON datasets USING GIN (tsv_document);

-- +goose Down
DROP INDEX IF EXISTS datasets_tsv_document_idx;

ALTER TABLE datasets
DROP COLUMN IF EXISTS "tsv_document";
