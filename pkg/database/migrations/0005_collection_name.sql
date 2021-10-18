-- +goose Up

DROP TRIGGER dataproduct_collections_set_modified ON dataproduct_collections;

ALTER TABLE dataproduct_collections
RENAME TO collections;

CREATE TRIGGER collections_set_modified
BEFORE UPDATE ON collections
FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();
