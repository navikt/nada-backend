-- +goose Up
CREATE TABLE IF NOT EXISTS "dashboards" (
    "id" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE "dashboards";