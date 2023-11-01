-- +goose Up
CREATE VIEW search
AS (
		SELECT
		"dp"."id" AS "element_id",
		'dataproduct' AS "element_type",
		coalesce("dp"."description", '') AS "description",
		"dp"."keywords",
		"dp"."group",
		"dp"."created",
		"dp"."last_modified",
		(
			setweight(to_tsvector('norwegian', "dp"."name"), 'A')
				|| setweight(to_tsvector('norwegian', coalesce("dp"."description", '')), 'B')
				|| setweight(to_tsvector('norwegian', coalesce(f_arr2text("dp"."keywords"), '')), 'C')
				|| setweight(to_tsvector('norwegian', coalesce("dp"."repo", '')), 'D')
				|| setweight(to_tsvector('norwegian', "dp"."type"::text), 'D')
				|| setweight(to_tsvector('norwegian', split_part(coalesce("dp"."group", ''), '@', 1)), 'D')
		) AS tsv_document,
		"tpm"."services"
	FROM "dataproducts" "dp"
	LEFT JOIN "third_party_mappings" "tpm" ON "tpm"."dataproduct_id" = "dp"."id"

	UNION

	SELECT
		"s"."id" AS "element_id",
		'story' AS "element_type",
		coalesce("s"."description", '') AS "description",
		'{}' AS "keywords",
		"s"."group",
		"s"."created",
		"s"."last_modified",
		(
			setweight(to_tsvector('norwegian', "s"."name"), 'A')
				|| setweight(to_tsvector('norwegian', coalesce("s"."description", '')), 'B')
				|| setweight(to_tsvector('norwegian', coalesce(f_arr2text("s"."keywords"), '')), 'C')
				|| setweight(to_tsvector('norwegian', split_part(coalesce("s"."group", ''), '@', 1)), 'D')
		) AS tsv_document,
		'{}' AS "services"
	FROM (
		SELECT "id", "name", "group", "created", "last_modified", "keywords",
		(
			SELECT string_agg("spec"->>'content', ' ')
			FROM (
				SELECT "spec"
				FROM "story_views"
				WHERE "story_id" = "story"."id"
				AND "type" IN ('markdown', 'header')
				ORDER BY "sort" ASC
			) "views"
		) AS "description"
		FROM "stories" "story"
	) "s"
);

-- +goose Down
DROP VIEW search;
