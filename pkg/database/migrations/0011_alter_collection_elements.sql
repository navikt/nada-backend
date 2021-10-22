-- +goose Up

DROP TABLE IF EXISTS collection_elements;

CREATE TABLE collection_elements
(
    element_id uuid NOT NULL,
    collection_id uuid NOT NULL,
    element_type TEXT NOT NULL
);
