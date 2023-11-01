-- +goose Up
CREATE TABLE insight_product (
    "id" uuid DEFAULT uuid_generate_v4(),
    "name" TEXT NOT NULL,
    "description" TEXT,
    "creator" TEXT NOT NULL,
    "created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "type" TEXT NOT NULL,
    "tsv_document" tsvector GENERATED ALWAYS AS (
        to_tsvector('norwegian', "name") || to_tsvector('norwegian', coalesce("description", ''))
    ) STORED,
    "link" TEXT NOT NULL,
    "keywords" TEXT [] NOT NULL DEFAULT '{}',
    "group" TEXT NOT NULL DEFAULT '',
    "teamkatalogen_url" TEXT,
    "product_area_id" TEXT,
    "team_id" TEXT,
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE insight_product;