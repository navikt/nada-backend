-- +goose Up
ALTER TABLE dataproduct_access RENAME TO dataset_access;
ALTER TABLE dataproduct_access_request RENAME TO dataset_access_requests;

-- +goose Down
ALTER TABLE dataset_access RENAME TO dataproduct_access;
ALTER TABLE dataset_access_requests RENAME TO dataproduct_access_request;