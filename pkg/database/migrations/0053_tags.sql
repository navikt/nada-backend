-- +goose Up
CREATE TABLE tags
(
    "id"            uuid                 DEFAULT uuid_generate_v4(),
    "text"          TEXT        NOT NULL,
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE tags;