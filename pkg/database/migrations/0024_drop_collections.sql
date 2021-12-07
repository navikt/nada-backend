-- +goose Up
DROP TRIGGER collections_set_modified ON collections;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS collection_elements;