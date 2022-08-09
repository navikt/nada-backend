-- +goose Up
ALTER TABLE dataproducts DROP COLUMN repo, DROP COLUMN keywords;

-- +goose Down
ALTER TABLE dataproducts ADD COLUMN repo TEXT, ADD COLUMN keywords TEXT[] NOT NULL DEFAULT '{}';