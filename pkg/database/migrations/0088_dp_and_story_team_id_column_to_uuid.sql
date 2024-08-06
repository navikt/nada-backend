-- +goose Up
DROP VIEW search;

DROP VIEW dataproduct_view;
DROP VIEW dataproduct_with_teamkatalogen_view;

ALTER TABLE dataproducts
    ALTER COLUMN team_id
        SET DATA TYPE UUID
        USING CASE
                  WHEN team_id ~* '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$' THEN team_id::UUID
        END;

CREATE VIEW dataproduct_with_teamkatalogen_view AS(
SELECT dp.*, tkt.name as team_name, tkpa.name as pa_name, tkt.product_area_id as pa_id FROM dataproducts dp LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON dp.team_id = tkt.id
);
CREATE VIEW dataproduct_view AS(
    SELECT
        dp.id as dp_id,
        dp.name as dp_name,
        dp.description as dp_description,
        dp.group as dp_group,
        dp.created as dp_created,
        dp.last_modified as dp_last_modified,
        dp.slug as dp_slug,
        dp.teamkatalogen_url as teamkatalogen_url,
        dp.team_contact as team_contact,
        dp.team_id as team_id,
		dp.team_name as team_name,
		dp.pa_name as pa_name,
        dp.pa_id as pa_id,
        ds.dataproduct_id as ds_dp_id,
        ds.id as ds_id,
        ds.name as ds_name,
        ds.description as ds_description,
        ds.created as ds_created,
        ds.last_modified as ds_last_modified,
        ds.slug as ds_slug,
        ds.keywords as ds_keywords
    FROM
        dataproduct_with_teamkatalogen_view dp
        LEFT JOIN datasets ds ON dp.id = ds.dataproduct_id
);

DROP VIEW story_with_teamkatalogen_view;

ALTER TABLE stories
    ALTER COLUMN team_id
        SET DATA TYPE UUID
        USING CASE
                  WHEN team_id ~* '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$' THEN team_id::UUID
        END;

CREATE VIEW story_with_teamkatalogen_view AS(
SELECT s.*, tkt.name as team_name, tkpa.name as pa_name FROM stories s LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON s.team_id = tkt.id
);

CREATE VIEW search AS (
    SELECT
        "dp"."id" AS "element_id",
        'dataproduct' AS "element_type",
        coalesce("dp"."description", '') AS "description",
        "dpk"."aggregated_keywords" AS "keywords",
        "dp"."group",
        "dp"."team_id",
        "dp"."created",
        "dp"."last_modified",
        (
            setweight(to_tsvector('norwegian', "dp"."name"), 'A') || setweight(
                to_tsvector('norwegian', coalesce("dp"."description", '')),
                'B'
            ) || setweight(
                to_tsvector(
                    'norwegian',
                    split_part(coalesce("dp"."group", ''), '@', 1)
                ),
                'D'
            )
        ) AS tsv_document,
        '{}' AS "services"
    FROM
        "dataproducts" "dp"
        LEFT JOIN (
            SELECT
                "dk"."dataproduct_id",
                coalesce(array_agg("flatterned_keywords_array"), '{}') as "aggregated_keywords"
            FROM
                (
                    SELECT
                        "dataproduct_id",
                        unnest("keywords") as "flatterned_keywords_array"
                    FROM
                        "datasets"
                ) as "dk"
            GROUP BY
                "dk"."dataproduct_id"
        ) AS "dpk" on "dp"."id" = "dataproduct_id"
    UNION
    SELECT
        "ds"."id" AS "element_id",
        'dataset' AS "element_type",
        coalesce("ds"."description", '') AS "description",
        "ds"."keywords",
        "dp"."group",
        "dp"."team_id",
        "ds"."created",
        "ds"."last_modified",
        (
            setweight(to_tsvector('norwegian', "ds"."name"), 'A') || setweight(
                to_tsvector('norwegian', coalesce("ds"."description", '')),
                'B'
            ) || setweight(
                to_tsvector(
                    'norwegian',
                    coalesce(f_arr2text("ds"."keywords"), '')
                ),
                'C'
            ) || setweight(
                to_tsvector('norwegian', coalesce("ds"."repo", '')),
                'D'
            ) || setweight(
                to_tsvector('norwegian', "ds"."type" :: text),
                'D'
            ) || setweight(
                to_tsvector(
                    'norwegian',
                    split_part(coalesce("dp"."group", ''), '@', 1)
                ),
                'D'
            )
        ) AS tsv_document,
        "tpm"."services"
    FROM
        "datasets" "ds"
        JOIN "dataproducts" "dp" ON "ds"."dataproduct_id" = "dp"."id"
        LEFT JOIN "third_party_mappings" "tpm" ON "tpm"."dataset_id" = "ds"."id"
)
UNION
SELECT
    "ss"."id" AS "element_id",
    'story' AS "element_type",
    "ss"."description" AS "description",
    "ss"."keywords" AS "keywords",
    "ss"."group" AS "group",
    "ss"."team_id",
    "ss"."created",
    "ss"."last_modified",
    (
        setweight(to_tsvector('norwegian', "ss"."name"), 'A') || setweight(
            to_tsvector('norwegian', "ss"."description"),
            'B'
        ) || setweight(
            to_tsvector(
                'norwegian',
                coalesce(f_arr2text("ss"."keywords"), '')
            ),
            'C'
        ) || setweight(
            to_tsvector(
                'norwegian',
                split_part(coalesce("ss"."creator", ''), '@', 1)
            ),
            'D'
        ) || setweight(
            to_tsvector(
                'norwegian',
                split_part(coalesce("ss"."group", ''), '@', 1)
            ),
            'D'
        )
    ) AS tsv_document,
    '{}' AS "services"
FROM
    "stories" ss;

-- +goose Down
DROP VIEW search;

DROP VIEW dataproduct_view;
DROP VIEW dataproduct_with_teamkatalogen_view;

ALTER TABLE dataproducts
    ALTER COLUMN team_id
        SET DATA TYPE TEXT
        USING team_id::TEXT;

CREATE VIEW dataproduct_with_teamkatalogen_view AS(
SELECT dp.*, tkt.name as team_name, tkpa.name as pa_name, tkt.product_area_id as pa_id FROM dataproducts dp LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON dp.team_id = tkt.id::TEXT
);
CREATE VIEW dataproduct_view AS(
    SELECT
        dp.id as dp_id,
        dp.name as dp_name,
        dp.description as dp_description,
        dp.group as dp_group,
        dp.created as dp_created,
        dp.last_modified as dp_last_modified,
        dp.slug as dp_slug,
        dp.teamkatalogen_url as teamkatalogen_url,
        dp.team_contact as team_contact,
        dp.team_id as team_id,
		dp.team_name as team_name,
		dp.pa_name as pa_name,
        dp.pa_id as pa_id,
        ds.dataproduct_id as ds_dp_id,
        ds.id as ds_id,
        ds.name as ds_name,
        ds.description as ds_description,
        ds.created as ds_created,
        ds.last_modified as ds_last_modified,
        ds.slug as ds_slug,
        ds.keywords as ds_keywords
    FROM
        dataproduct_with_teamkatalogen_view dp
        LEFT JOIN datasets ds ON dp.id = ds.dataproduct_id
);

DROP VIEW story_with_teamkatalogen_view;

ALTER TABLE stories
    ALTER COLUMN team_id
        SET DATA TYPE TEXT
        USING team_id::TEXT;

CREATE VIEW story_with_teamkatalogen_view AS(
SELECT s.*, tkt.name as team_name, tkpa.name as pa_name FROM stories s LEFT JOIN 
	(tk_teams tkt LEFT JOIN tk_product_areas tkpa
	ON tkt.product_area_id = tkpa.id)
	ON s.team_id = tkt.id::TEXT
);

CREATE VIEW search AS (
    SELECT
        "dp"."id" AS "element_id",
        'dataproduct' AS "element_type",
        coalesce("dp"."description", '') AS "description",
        "dpk"."aggregated_keywords" AS "keywords",
        "dp"."group",
        "dp"."team_id",
        "dp"."created",
        "dp"."last_modified",
        (
            setweight(to_tsvector('norwegian', "dp"."name"), 'A') || setweight(
                to_tsvector('norwegian', coalesce("dp"."description", '')),
                'B'
            ) || setweight(
                to_tsvector(
                    'norwegian',
                    split_part(coalesce("dp"."group", ''), '@', 1)
                ),
                'D'
            )
        ) AS tsv_document,
        '{}' AS "services"
    FROM
        "dataproducts" "dp"
        LEFT JOIN (
            SELECT
                "dk"."dataproduct_id",
                coalesce(array_agg("flatterned_keywords_array"), '{}') as "aggregated_keywords"
            FROM
                (
                    SELECT
                        "dataproduct_id",
                        unnest("keywords") as "flatterned_keywords_array"
                    FROM
                        "datasets"
                ) as "dk"
            GROUP BY
                "dk"."dataproduct_id"
        ) AS "dpk" on "dp"."id" = "dataproduct_id"
    UNION
    SELECT
        "ds"."id" AS "element_id",
        'dataset' AS "element_type",
        coalesce("ds"."description", '') AS "description",
        "ds"."keywords",
        "dp"."group",
        "dp"."team_id",
        "ds"."created",
        "ds"."last_modified",
        (
            setweight(to_tsvector('norwegian', "ds"."name"), 'A') || setweight(
                to_tsvector('norwegian', coalesce("ds"."description", '')),
                'B'
            ) || setweight(
                to_tsvector(
                    'norwegian',
                    coalesce(f_arr2text("ds"."keywords"), '')
                ),
                'C'
            ) || setweight(
                to_tsvector('norwegian', coalesce("ds"."repo", '')),
                'D'
            ) || setweight(
                to_tsvector('norwegian', "ds"."type" :: text),
                'D'
            ) || setweight(
                to_tsvector(
                    'norwegian',
                    split_part(coalesce("dp"."group", ''), '@', 1)
                ),
                'D'
            )
        ) AS tsv_document,
        "tpm"."services"
    FROM
        "datasets" "ds"
        JOIN "dataproducts" "dp" ON "ds"."dataproduct_id" = "dp"."id"
        LEFT JOIN "third_party_mappings" "tpm" ON "tpm"."dataset_id" = "ds"."id"
)
UNION
SELECT
    "ss"."id" AS "element_id",
    'story' AS "element_type",
    "ss"."description" AS "description",
    "ss"."keywords" AS "keywords",
    "ss"."group" AS "group",
    "ss"."team_id",
    "ss"."created",
    "ss"."last_modified",
    (
        setweight(to_tsvector('norwegian', "ss"."name"), 'A') || setweight(
            to_tsvector('norwegian', "ss"."description"),
            'B'
        ) || setweight(
            to_tsvector(
                'norwegian',
                coalesce(f_arr2text("ss"."keywords"), '')
            ),
            'C'
        ) || setweight(
            to_tsvector(
                'norwegian',
                split_part(coalesce("ss"."creator", ''), '@', 1)
            ),
            'D'
        ) || setweight(
            to_tsvector(
                'norwegian',
                split_part(coalesce("ss"."group", ''), '@', 1)
            ),
            'D'
        )
    ) AS tsv_document,
    '{}' AS "services"
FROM
    "stories" ss;
