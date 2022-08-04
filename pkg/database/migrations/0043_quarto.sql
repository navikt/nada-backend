-- +goose Up
CREATE TABLE "quarto" (
    "id"      uuid          DEFAULT uuid_generate_v4(),
    "team"    TEXT NOT NULL,
    "content" TEXT NOT NULL,
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE "quarto";
