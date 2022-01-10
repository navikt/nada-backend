-- +goose Up
CREATE OR REPLACE FUNCTION create_graph_uri_draft_reference() RETURNS TRIGGER AS
$$ BEGIN IF (NEW.type = 'plotly') THEN INSERT INTO story_view_drafts ("story_id", "sort", "type", "spec") VALUES (NEW.story_id, NEW.sort, 'graph_uri', CONCAT('{"id": "', NEW.ID, '"}')::jsonb); END IF; RETURN NULL; END; $$
language plpgsql;

CREATE TRIGGER graph_view_draft_create_reference
    AFTER INSERT
    ON story_view_drafts
    FOR EACH ROW
EXECUTE PROCEDURE create_graph_uri_draft_reference();

CREATE OR REPLACE FUNCTION create_graph_uri_reference() RETURNS TRIGGER AS
$$ BEGIN IF (NEW.type = 'plotly') THEN INSERT INTO story_views ("story_id", "sort", "type", "spec") VALUES (NEW.story_id, NEW.sort, 'graph_uri', CONCAT('{"id": "', NEW.ID, '"}')::jsonb); END IF; RETURN NULL; END; $$
language plpgsql;

CREATE TRIGGER graph_view_create_reference
    AFTER INSERT
    ON story_views
    FOR EACH ROW
EXECUTE PROCEDURE create_graph_uri_reference();
