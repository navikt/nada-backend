-- +goose Up
ALTER TABLE dataset_access ADD COLUMN "owner" TEXT;

UPDATE dataset_access SET "owner" = SPLIT_PART(existing_dataset_access.subject, ':', 2)
FROM (SELECT "id", "subject" FROM dataset_access) AS existing_dataset_access
WHERE dataset_access.id = existing_dataset_access.id;

ALTER TABLE dataset_access ALTER COLUMN "owner" SET NOT NULL;

-- +goose Down
ALTER TABLE dataset_access DROP COLUMN "owner";
