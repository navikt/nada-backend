-- +goose Up
ALTER TABLE
    joinable_views_datasource DROP CONSTRAINT IF EXISTS fk_datasource_id;

ALTER TABLE
    datasource_bigquery DROP CONSTRAINT IF EXISTS fk_bigquery_dataset;

ALTER TABLE
    joinable_views_datasource
ADD 
    COLUMN "deleted" TIMESTAMPTZ;

CREATE
OR REPLACE FUNCTION set_joinable_view_deleted_on_dataset_deleted() RETURNS TRIGGER AS $$ BEGIN UPDATE joinable_views_datasource SET deleted = NOW() FROM datasets INNER JOIN datasource_bigquery ON datasets.id = datasource_bigquery.dataset_id WHERE datasource_id = datasource_bigquery.id AND datasets.id = OLD.id; RETURN OLD; END; $$
LANGUAGE plpgsql;

CREATE TRIGGER dataset_deleted BEFORE DELETE ON datasets FOR EACH ROW EXECUTE PROCEDURE set_joinable_view_deleted_on_dataset_deleted();

-- +goose Down
DROP TRIGGER IF EXISTS dataset_deleted ON datasets;

DROP FUNCTION IF EXISTS set_joinable_view_deleted_on_datasource_deleted;

ALTER TABLE
    joinable_views_datasource DROP COLUMN IF EXISTS "deleted";

ALTER TABLE
    joinable_views_datasource
ADD 
    CONSTRAINT fk_datasource_id FOREIGN KEY (datasource_id) REFERENCES datasource_bigquery (id) ON DELETE CASCADE;

ALTER TABLE
    datasource_bigquery
ADD 
    CONSTRAINT fk_bigquery_dataset FOREIGN KEY (dataset_id) REFERENCES datasets (id) ON DELETE CASCADE;