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
