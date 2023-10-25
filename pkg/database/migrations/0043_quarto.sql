-- +goose Up
CREATE TABLE "quarto" (
    "id"            uuid                  DEFAULT uuid_generate_v4(),
    "owner"         TEXT        NOT NULL,
    "created"       TIMESTAMPTZ NOT NULL  DEFAULT NOW(),
    "last_modified" TIMESTAMPTZ NOT NULL  DEFAULT NOW(),
    "keywords"      TEXT[]      NOT NULL  DEFAULT '{}',
    "content"       TEXT        NOT NULL,
    PRIMARY KEY (id)
);

CREATE TRIGGER quarto_set_modified
    BEFORE UPDATE
    ON quarto
    FOR EACH ROW
EXECUTE PROCEDURE update_modified_timestamp();

-- +goose Down
DROP TRIGGER quarto_set_modified ON quarto;
DROP TABLE "quarto";
