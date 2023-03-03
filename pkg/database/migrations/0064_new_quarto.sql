-- +goose Up
DROP TRIGGER quarto_set_modified ON quarto;
DROP TABLE "quarto";

CREATE TABLE quarto_stories (
	"id" uuid DEFAULT uuid_generate_v4(),
	"name" TEXT NOT NULL,
    "creator" TEXT NOT NULL,
	"created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	"description" TEXT NOT NULL,
    "keywords"      TEXT[]      NOT NULL  DEFAULT '{}',
    "teamkatalogen_url" TEXT,
    "product_area_id" TEXT,
    "team_id" TEXT,
	PRIMARY KEY (id)
);

CREATE TRIGGER quarto_story_modified
    BEFORE UPDATE
    ON quarto_stories
    FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

-- +goose Down
DROP TRIGGER quarto_story_modified ON quarto_stories;
DROP TABLE "quarto_stories";

CREATE TABLE "quarto" (
    "id"            uuid                  DEFAULT uuid_generate_v4(),
    "owner"         TEXT        NOT NULL,
    "created"       TIMESTAMPTZ NOT NULL  DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL  DEFAULT NOW(),
    "keywords"      TEXT[]      NOT NULL  DEFAULT '{}',
    "content"       TEXT        NOT NULL,
    PRIMARY KEY (id)
);

CREATE TRIGGER quarto_set_modified
    BEFORE UPDATE
    ON quarto
    FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();
