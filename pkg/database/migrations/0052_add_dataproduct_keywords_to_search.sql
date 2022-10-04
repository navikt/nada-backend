-- +goose Up
DROP VIEW search;

CREATE VIEW search
AS (
   SELECT
       "dp"."id" AS "element_id",
       'dataproduct' AS "element_type",
       coalesce("dp"."description", '') AS "description",
       "dpk"."aggregated_keywords" AS "keywords",
       "dp"."group",
       "dp"."created",
       "dp"."last_modified",
       (
                   setweight(to_tsvector('norwegian', "dp"."name"), 'A')
                   || setweight(to_tsvector('norwegian', coalesce("dp"."description", '')), 'B')
               || setweight(to_tsvector('norwegian', split_part(coalesce("dp"."group", ''), '@', 1)), 'D')
           ) AS tsv_document,
       '{}' AS "services"
   FROM "dataproducts" "dp" LEFT JOIN
        (
            SELECT "dataproduct_id", coalesce(array_agg("flatterned_keywords_array"), '{}') as "aggregated_keywords"
            FROM
            (
                SELECT "dataproduct_id", unnest("keywords") as "flatterned_keywords_array" FROM "datasets"
            ) as "dk"
            GROUP BY "dataproduct_id"
        ) AS "dpk" on "dp"."id" = "dataproduct_id"

   UNION

   SELECT
       "ds"."id" AS "element_id",
       'dataset' AS "element_type",
       coalesce("ds"."description", '') AS "description",
       "ds"."keywords",
       "dp"."group",
       "ds"."created",
       "ds"."last_modified",
       (
                               setweight(to_tsvector('norwegian', "ds"."name"), 'A')
                               || setweight(to_tsvector('norwegian', coalesce("ds"."description", '')), 'B')
                           || setweight(to_tsvector('norwegian', coalesce(f_arr2text("ds"."keywords"), '')), 'C')
                       || setweight(to_tsvector('norwegian', coalesce("ds"."repo", '')), 'D')
                   || setweight(to_tsvector('norwegian', "ds"."type"::text), 'D')
               || setweight(to_tsvector('norwegian', split_part(coalesce("dp"."group", ''), '@', 1)), 'D')
           ) AS tsv_document,
       "tpm"."services"
   FROM "datasets" "ds"
            JOIN "dataproducts" "dp" ON "ds"."dataproduct_id" = "dp"."id"
            LEFT JOIN "third_party_mappings" "tpm" ON "tpm"."dataset_id" = "ds"."id"

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

CREATE VIEW search 
AS (
    SELECT
    	"dp"."id" AS "element_id",
		'dataproduct' AS "element_type",
		coalesce("dp"."description", '') AS "description",
		'{}' AS "keywords",
		"dp"."group",
		"dp"."created",
		"dp"."last_modified",
		(
			setweight(to_tsvector('norwegian', "dp"."name"), 'A')
				|| setweight(to_tsvector('norwegian', coalesce("dp"."description", '')), 'B')
				|| setweight(to_tsvector('norwegian', split_part(coalesce("dp"."group", ''), '@', 1)), 'D')
		) AS tsv_document,
		'{}' AS "services"
	FROM "dataproducts" "dp"
	
    UNION

	SELECT
		"ds"."id" AS "element_id",
		'dataset' AS "element_type",
		coalesce("ds"."description", '') AS "description",
		"ds"."keywords",
        "dp"."group",
		"ds"."created",
		"ds"."last_modified",
		(
			setweight(to_tsvector('norwegian', "ds"."name"), 'A')
				|| setweight(to_tsvector('norwegian', coalesce("ds"."description", '')), 'B')
				|| setweight(to_tsvector('norwegian', coalesce(f_arr2text("ds"."keywords"), '')), 'C')
				|| setweight(to_tsvector('norwegian', coalesce("ds"."repo", '')), 'D')
				|| setweight(to_tsvector('norwegian', "ds"."type"::text), 'D')
				|| setweight(to_tsvector('norwegian', split_part(coalesce("dp"."group", ''), '@', 1)), 'D')
		) AS tsv_document,
		"tpm"."services"
	FROM "datasets" "ds"
    JOIN "dataproducts" "dp" ON "ds"."dataproduct_id" = "dp"."id"
	LEFT JOIN "third_party_mappings" "tpm" ON "tpm"."dataset_id" = "ds"."id"

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