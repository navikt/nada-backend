-- +goose Up

ALTER TABLE collections DROP COLUMN repo;
