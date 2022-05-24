-- +goose Up
ALTER TABLE dataproduct_access_request ADD COLUMN reason TEXT;

-- +goose Down
ALTER TABLE dataproduct_access_request DROP COLUMN reason;