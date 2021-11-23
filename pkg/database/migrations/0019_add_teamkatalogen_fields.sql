-- +goose Up

ALTER TABLE dataproducts 
ADD COLUMN "teamkatalogen_url" TEXT
;

ALTER TABLE collections
ADD COLUMN "teamkatalogen_url" TEXT
;