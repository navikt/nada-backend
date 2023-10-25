-- +goose Up
ALTER TABLE quarto_stories ADD COLUMN "group" TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE quarto_stories DROP COLUMN "group";