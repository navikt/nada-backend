-- +goose Up

ALTER TABLE dataproducts ALTER COLUMN slug SET NOT NULL;
