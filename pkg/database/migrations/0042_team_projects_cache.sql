-- +goose Up
CREATE TABLE "team_projects" (
    "team" TEXT NOT NULL,
    "project" TEXT NOT NULL,
    PRIMARY KEY (team)
);

-- +goose Down
DROP TABLE "team_projects";
