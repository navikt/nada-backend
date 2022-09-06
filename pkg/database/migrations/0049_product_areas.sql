-- +goose Up
CREATE TABLE IF NOT EXISTS "product_areas" (
    "id" uuid NOT NULL DEFAULT uuid_generate_v4(),
    "external_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    PRIMARY KEY (external_id)
);

-- +goose Down
DROP TABLE "product_areas";
