-- +goose Up
CREATE TABLE tags
(
    "id"            uuid        DEFAULT uuid_generate_v4(),
    "phrase"        TEXT        UNIQUE NOT NULL,
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE tags;