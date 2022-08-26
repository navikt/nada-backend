-- +goose Up
DROP VIEW search;

ALTER TABLE third_party_mappings 
    DROP CONSTRAINT fk_tpm_bigquery_dataproduct;
ALTER TABLE third_party_mappings
    ADD COLUMN dataset_id uuid;
UPDATE third_party_mappings a SET dataset_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);
ALTER TABLE third_party_mappings
    ALTER COLUMN dataset_id SET NOT NULL;
ALTER TABLE third_party_mappings DROP CONSTRAINT third_party_mappings_pkey;
ALTER TABLE third_party_mappings ADD PRIMARY KEY (dataset_id);

ALTER TABLE third_party_mappings
    DROP COLUMN dataproduct_id;
ALTER TABLE third_party_mappings
    ADD CONSTRAINT fk_tpm_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;    

ALTER TABLE dataset_access 
    DROP CONSTRAINT fk_access_dataproduct;
UPDATE dataset_access a SET dataproduct_id = (SELECT id FROM datasets WHERE dataproduct_id = a.dataproduct_id);
ALTER TABLE dataset_access
    RENAME COLUMN dataproduct_id TO dataset_id;
ALTER TABLE dataset_access
    ADD CONSTRAINT fk_access_dataset FOREIGN KEY (dataset_id)
            REFERENCES datasets (id) ON DELETE CASCADE;

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


-- +goose Down

DROP VIEW search;

ALTER TABLE third_party_mappings
    DROP CONSTRAINT fk_tpm_dataset;
ALTER TABLE third_party_mappings
    ADD COLUMN dataproduct_id uuid;
UPDATE third_party_mappings SET dataproduct_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = dataset_id);
ALTER TABLE third_party_mappings
    ALTER COLUMN dataproduct_id SET NOT NULL;
ALTER TABLE third_party_mappings DROP CONSTRAINT third_party_mappings_pkey;
ALTER TABLE third_party_mappings ADD PRIMARY KEY (dataproduct_id);

ALTER TABLE third_party_mappings
    DROP COLUMN dataset_id;
ALTER TABLE third_party_mappings
    ADD CONSTRAINT fk_tpm_bigquery_dataproduct FOREIGN KEY (dataproduct_id)
        REFERENCES dataproducts (id) ON DELETE CASCADE;

ALTER TABLE dataset_access 
    DROP CONSTRAINT fk_access_dataset;
UPDATE dataset_access SET dataset_id = (SELECT dataproduct_id FROM datasets WHERE dataset_id = dataset_id);
ALTER TABLE dataset_access
    RENAME COLUMN dataset_id TO dataproduct_id;
ALTER TABLE dataset_access
    ADD CONSTRAINT fk_access_dataproduct FOREIGN KEY (dataproduct_id)
            REFERENCES dataproducts (id) ON DELETE CASCADE;

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
