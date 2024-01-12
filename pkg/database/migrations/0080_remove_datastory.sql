-- +goose Up
DROP TRIGGER story_token ON stories;
DROP FUNCTION create_story_token();
DROP TABLE story_tokens;

DROP TABLE story_view_drafts;
DROP TABLE story_drafts;
DROP TABLE story_views;
DROP TRIGGER stories_set_modified on stories;
DROP TABLE stories;
DROP TYPE story_view_type;

ALTER TABLE quarto_stories RENAME TO stories;

DROP VIEW search;

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
ALTER TABLE stories RENAME TO quarto_stories;

CREATE TYPE story_view_type AS ENUM ('markdown', 'header', 'plotly', 'vega');

CREATE TABLE stories (
	"id" uuid DEFAULT uuid_generate_v4(),
	"name" TEXT NOT NULL,
	"created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "description" TEXT,
    "keywords" TEXT[] NOT NULL DEFAULT '{}',
    "teamkatalogen_url" TEXT,
    "team_id" TEXT,
    "last_modified" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"group" TEXT NOT NULL,
	PRIMARY KEY (id)
);

CREATE TRIGGER stories_set_modified
    BEFORE UPDATE
    ON stories
    FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

CREATE TABLE story_views (
	"id" uuid DEFAULT uuid_generate_v4(),
	"story_id" uuid NOT NULL,
	"sort" INT NOT NULL DEFAULT 0,
	"type" story_view_type NOT NULL,
	"spec" JSONB NOT NULL,
	PRIMARY KEY (id),
	CONSTRAINT fk_story_views_story
			FOREIGN KEY (story_id)
					REFERENCES stories (id) ON DELETE CASCADE
);

CREATE TABLE story_drafts (
	"id" uuid DEFAULT uuid_generate_v4(),
	"name" TEXT NOT NULL,
	"created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (id)
);

CREATE TABLE story_view_drafts (
	"id" uuid DEFAULT uuid_generate_v4(),
	"story_id" uuid NOT NULL,
	"sort" INT NOT NULL DEFAULT 0,
	"type" story_view_type NOT NULL,
	"spec" JSONB NOT NULL,
	PRIMARY KEY (id),
	CONSTRAINT fk_story_view_drafts_story
			FOREIGN KEY (story_id)
					REFERENCES story_drafts (id) ON DELETE CASCADE
);

CREATE TABLE story_tokens (
    "id" uuid DEFAULT uuid_generate_v4(),
	"story_id" uuid NOT NULL,
	"token" uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE,
	PRIMARY KEY (id),
	CONSTRAINT fk_story_views_story
		FOREIGN KEY (story_id)
			REFERENCES stories (id) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION create_story_token() RETURNS TRIGGER AS
$$ BEGIN INSERT INTO story_tokens ("story_id") VALUES (NEW.id); RETURN NULL; END; $$
language plpgsql;

CREATE TRIGGER story_token
    AFTER INSERT
    ON stories
    FOR EACH ROW
EXECUTE PROCEDURE create_story_token();

DROP VIEW search;

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
    "qs"."id" AS "element_id",
    'quarto_story' AS "element_type",
    "qs"."description" AS "description",
    "qs"."keywords" AS "keywords",
    "qs"."group" AS "group",
    "qs"."team_id",
    "qs"."created",
    "qs"."last_modified",
    (
        setweight(to_tsvector('norwegian', "qs"."name"), 'A') || setweight(
            to_tsvector('norwegian', "qs"."description"),
            'B'
        ) || setweight(
            to_tsvector(
                'norwegian',
                coalesce(f_arr2text("qs"."keywords"), '')
            ),
            'C'
        ) || setweight(
            to_tsvector(
                'norwegian',
                split_part(coalesce("qs"."creator", ''), '@', 1)
            ),
            'D'
        ) || setweight(
            to_tsvector(
                'norwegian',
                split_part(coalesce("qs"."group", ''), '@', 1)
            ),
            'D'
        )
    ) AS tsv_document,
    '{}' AS "services"
FROM
    "quarto_stories" qs;
