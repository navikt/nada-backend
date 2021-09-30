-- +goose Up
CREATE TABLE dataproducts (
    "id" uuid DEFAULT uuid_generate_v4(),
    "name" TEXT NOT NULL,
    "description" TEXT,
    "slug" TEXT NOT NULL,
    "repo" TEXT,
    "created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "team" TEXT not null,
    "keywords" TEXT[],
    PRIMARY KEY(id)
);

CREATE TRIGGER dataproducts_set_modified
BEFORE UPDATE ON dataproducts
FOR EACH ROW
EXECUTE PROCEDURE moddatetime(last_modified);

-- +goose Down
DROP TABLE dataproducts;
DROP TRIGGER dataproducts_set_modified;
