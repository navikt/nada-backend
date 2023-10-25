-- +goose Up
CREATE TABLE http_cache (
  id SERIAL PRIMARY KEY,
  endpoint TEXT UNIQUE NOT NULL,
  response_body BYTEA NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  last_tried_update_at TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose Down
DROP TABLE http_cache;