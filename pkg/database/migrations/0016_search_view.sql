-- +goose Up

DROP VIEW IF EXISTS search;

CREATE OR REPLACE VIEW search
AS (
	SELECT
		id AS element_id,
		'dataproduct' AS element_type,
		"description",
		last_modified,
		keywords,
		"group",
		(
			setweight(to_tsvector('norwegian', "name"), 'A')
				|| setweight(to_tsvector('norwegian', coalesce("description", '')), 'B')
				|| setweight(to_tsvector('norwegian', coalesce(f_arr2text("keywords"), '')), 'C')
				|| setweight(to_tsvector('norwegian', coalesce("repo", '')), 'D')
				|| setweight(to_tsvector('norwegian', "type"::text), 'D')
				|| setweight(to_tsvector('norwegian', split_part(coalesce("group", ''), '@', 1)), 'D')
		) AS tsv_document
	FROM dataproducts

	UNION ALL

	SELECT
		id AS element_id,
		'collection' AS element_type,
		"description",
		last_modified,
		keywords,
		"group",
		(
			setweight(to_tsvector('norwegian', "name"), 'A')
				|| setweight(to_tsvector('norwegian', coalesce("description", '')), 'B')
				|| setweight(to_tsvector('norwegian', coalesce(f_arr2text("keywords"), '')), 'C')
				|| setweight(to_tsvector('norwegian', split_part(coalesce("group", ''), '@', 1)), 'D')
		) AS tsv_document
	FROM collections
);
