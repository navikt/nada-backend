-- +goose Up
CREATE TABLE "team_projects" (
    "team" TEXT NOT NULL,
    "project" TEXT NOT NULL
);

-- +goose Down
DROP TABLE "team_projects";
