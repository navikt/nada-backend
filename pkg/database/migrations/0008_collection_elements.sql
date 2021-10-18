-- +goose Up

CREATE TABLE collection_elements
(
    element_id TEXT NOT NULL,
    collection_id TEXT NOT NULL,
    element_type TEXT NOT NULL
);
