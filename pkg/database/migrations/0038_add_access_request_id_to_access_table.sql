-- +goose Up
ALTER TABLE dataproduct_access ADD COLUMN access_request_id uuid;

-- +goose Down
ALTER TABLE dataproduct_access DROP COLUMN access_request_id;