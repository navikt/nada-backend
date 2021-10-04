-- +goose Up
CREATE OR REPLACE FUNCTION f_arr2text(text[]) 
  RETURNS text LANGUAGE sql IMMUTABLE AS $$SELECT array_to_string($1, ',')$$;

ALTER TABLE dataproducts
ADD COLUMN "tsv_document" tsvector GENERATED ALWAYS AS (
	   to_tsvector('norwegian', "name")
	|| to_tsvector('norwegian', coalesce("description", ''))
	|| to_tsvector('norwegian', coalesce(f_arr2text("keywords"), ''))
	|| to_tsvector('norwegian', coalesce("team", ''))
) STORED;

CREATE INDEX dataproducts_tsv_document_idx ON dataproducts USING GIN (tsv_document);

-- +goose Down
DROP INDEX IF EXISTS dataproducts_tsv_document_idx;

ALTER TABLE dataproducts
DROP COLUMN IF EXISTS "tsv_document";
