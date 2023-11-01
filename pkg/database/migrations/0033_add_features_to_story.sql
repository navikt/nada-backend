-- +goose Up
ALTER TABLE stories
    ADD COLUMN "description"   TEXT,
    ADD COLUMN "keywords"      TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE stories
    DROP COLUMN "description",
    DROP COLUMN "keywords";
