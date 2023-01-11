-- +goose Up
ALTER TABLE datasets ADD COLUMN target_user TEXT ;

-- +goose Down
ALTER TABLE dataset DROP COLUMN target_user;