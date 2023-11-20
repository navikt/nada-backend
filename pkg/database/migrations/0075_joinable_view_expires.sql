-- +goose Up
ALTER TABLE joinable_views ADD COLUMN "expires" TIMESTAMPTZ;
ALTER TABLE joinable_views ADD COLUMN "deleted" TIMESTAMPTZ;

-- +goose Down
ALTER TABLE joinable_views DROP COLUMN "expires";
ALTER TABLE joinable_views DROP COLUMN "deleted";