-- +goose Up
CREATE TYPE story_view_type AS ENUM ('markdown', 'header', 'plotly');

CREATE TABLE stories (
	"id" uuid DEFAULT uuid_generate_v4(),
	"name" TEXT NOT NULL,
	"created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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

-- +goose Down
DROP TABLE story_view_drafts;
DROP TABLE story_drafts;
DROP TABLE story_views;
DROP TRIGGER stories_set_modified on stories;
DROP TABLE stories;
DROP TYPE story_view_type;
