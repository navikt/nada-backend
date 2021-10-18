-- +goose Up

ALTER TABLE dataproducts 
ADD COLUMN "slug" TEXT,
ADD COLUMN "repo" TEXT,
ADD COLUMN "keywords" TEXT[] NOT NULL DEFAULT '{}'
;
