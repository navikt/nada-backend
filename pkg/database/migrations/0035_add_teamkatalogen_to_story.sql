-- +goose Up
ALTER TABLE stories ADD COLUMN "teamkatalogen_url" TEXT;

-- +goose Down
ALTER TABLE stories DROP COLUMN "teamkatalogen_url";
