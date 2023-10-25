-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

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

-- +goose Down
DROP TRIGGER story_token ON stories;
DROP FUNCTION create_story_token();
DROP TABLE story_tokens;
DROP EXTENSION "pgcrypto";
