-- +goose Up
ALTER TABLE datasource_bigquery ADD COLUMN "description" TEXT;