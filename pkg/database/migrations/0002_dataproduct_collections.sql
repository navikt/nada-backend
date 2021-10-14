-- +goose Up
CREATE TABLE dataproduct_collections (
    "id" uuid DEFAULT uuid_generate_v4(),
    "name" TEXT NOT NULL,
    "description" TEXT,
    "slug" TEXT NOT NULL,
    "repo" TEXT,
    "created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "team" TEXT not null,
    "keywords" TEXT[],
    "tsv_document"  tsvector GENERATED ALWAYS AS (
                    to_tsvector('norwegian', "name")
                    || to_tsvector('norwegian', coalesce("description", ''))
                || to_tsvector('norwegian', coalesce(f_arr2text("keywords"), ''))
            || to_tsvector('norwegian', coalesce("team", ''))
        ) STORED,
    PRIMARY KEY(id)
);

CREATE TRIGGER dataproduct_collections_set_modified
BEFORE UPDATE ON dataproduct_collections
FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

-- +goose Down
DROP TABLE dataproduct_collections;
DROP TRIGGER dataproduct_collections_set_modified;
