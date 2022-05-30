-- +goose Up
ALTER TABLE dataproduct_requesters RENAME TO dataset_requesters;

-- +goose Down

ALTER TABLE dataset_requesters RENAME TO dataproduct_requesters;