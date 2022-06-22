-- +goose Up
CREATE EXTENSION pg_trgm;

-- +goose Down
DROP EXTENSION pg_trgm;
