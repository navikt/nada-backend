-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE TABLE nada_tokens (
	"team" TEXT NOT NULL,
	"token" uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE,
	PRIMARY KEY (team)
);

CREATE OR REPLACE FUNCTION create_nada_token() RETURNS TRIGGER AS
$$ BEGIN INSERT INTO nada_tokens ("team") VALUES (NEW.team) ON CONFLICT DO NOTHING; RETURN NULL; END; $$
language plpgsql;

CREATE TRIGGER nada_token
    AFTER INSERT
    ON team_projects
    FOR EACH ROW
EXECUTE PROCEDURE create_nada_token();

-- +goose Down
DROP TRIGGER nada_token ON team_projects;
DROP FUNCTION create_nada_token();
DROP TABLE nada_tokens;
DROP EXTENSION "pgcrypto";
