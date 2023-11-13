-- +goose Up
ALTER TABLE joinable_views_datasource DROP CONSTRAINT fk_datasource_id;
ALTER TABLE joinable_views_datasource ADD COLUMN "deleted" TIMESTAMPTZ;

CREATE OR REPLACE FUNCTION set_joinable_view_deleted_on_datasource_deleted() RETURNS TRIGGER AS
$$ BEGIN UPDATE joinable_views_datasource SET deleted = NOW() WHERE datasource_id = OLD.id; RETURN OLD; END; $$
LANGUAGE plpgsql;

CREATE TRIGGER joinable_view_deleted
    BEFORE DELETE
    ON datasource_bigquery
    FOR EACH ROW
EXECUTE PROCEDURE set_joinable_view_deleted_on_datasource_deleted();

-- +goose Down
DROP TRIGGER joinable_view_deleted ON datasource_bigquery;
DROP FUNCTION set_joinable_view_deleted_on_datasource_deleted;
ALTER TABLE joinable_views_datasource DROP COLUMN "deleted";
ALTER TABLE joinable_views_datasource ADD CONSTRAINT fk_datasource_id FOREIGN KEY (datasource_id) REFERENCES datasource_bigquery (id) ON DELETE CASCADE;