-- +goose Up
ALTER TABLE datasets
    ADD COLUMN anonymisation_description TEXT;

-- +goose Down
ALTER TABLE datasets 
    DROP COLUMN anonymisation_description;