-- +goose Up
DROP VIEW IF EXISTS search;

CREATE VIEW search
AS (
	SELECT
		id AS element_id,
		'dataproduct' AS element_type,
		last_modified,
		keywords,
		"group",
		created,
		(
			to_tsvector('norwegian', "name")
				|| to_tsvector('norwegian', coalesce("description", ''))
				|| to_tsvector('norwegian', coalesce(f_arr2text("keywords"), ''))
				|| to_tsvector('norwegian', coalesce("repo", ''))
				|| to_tsvector('norwegian', "type"::text)
				|| to_tsvector('norwegian', split_part(coalesce("group", ''), '@', 1))
		) AS tsv_document
	FROM dataproducts

	UNION ALL

	SELECT
		id AS element_id,
		'collection' AS element_type,
		last_modified,
		keywords,
		"group",
		created,
		(
			to_tsvector('norwegian', "name")
				|| to_tsvector('norwegian', coalesce("description", ''))
				|| to_tsvector('norwegian', coalesce(f_arr2text("keywords"), ''))
				|| to_tsvector('norwegian', split_part(coalesce("group", ''), '@', 1))
		) AS tsv_document
	FROM collections
);
