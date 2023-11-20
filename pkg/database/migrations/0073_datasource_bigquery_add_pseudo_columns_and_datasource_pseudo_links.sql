-- +goose Up
-- Step 1: Add the new column with UUID data type
ALTER TABLE
    datasource_bigquery
ADD
    COLUMN id UUID NOT NULL DEFAULT uuid_generate_v4(),
ADD
    COLUMN "is_reference" BOOLEAN NOT NULL DEFAULT FALSE,
ADD
    COLUMN "pseudo_columns" TEXT [] DEFAULT ARRAY [] :: text [];

;

-- Step 3: Drop the existing primary key constraint
ALTER TABLE
    datasource_bigquery DROP CONSTRAINT datasource_bigquery_pkey;

-- Step 4: Create a new primary key constraint on ID1
ALTER TABLE
    datasource_bigquery
ADD
    PRIMARY KEY (id);

-- End the transaction
-- +goose Down
-- Start a transaction
ALTER TABLE
    datasource_bigquery DROP CONSTRAINT datasource_bigquery_pkey;

ALTER TABLE
    datasource_bigquery
ADD
    PRIMARY KEY (dataset_id);

ALTER TABLE
    datasource_bigquery DROP COLUMN "id",
    DROP COLUMN "is_reference",
    DROP COLUMN "pseudo_columns";