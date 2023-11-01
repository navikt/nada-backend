-- +goose Up
CREATE TABLE joinable_views (
    "id" uuid DEFAULT uuid_generate_v4(),
    "owner" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "created" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE TABLE joinable_views_datasource (
    "id" uuid DEFAULT uuid_generate_v4(),
    "joinable_view_id" uuid NOT NULL,
    "datasource_id" uuid NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_joinable_views FOREIGN KEY (joinable_view_id) REFERENCES joinable_views (id) ON DELETE CASCADE,
    CONSTRAINT fk_datasource_id FOREIGN KEY (datasource_id) REFERENCES datasource_bigquery (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE joinable_views_datasource;

DROP TABLE joinable_views;