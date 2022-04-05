-- +goose Up
ALTER TABLE stories ADD COLUMN "teamkatalogen_url" TEXT;
