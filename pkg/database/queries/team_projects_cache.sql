-- name: AddTeamProject :one
INSERT INTO team_projects ("team",
                           "project")
VALUES (
    @team,
    @project
)
RETURNING *;

-- name: GetTeamProjects :many
SELECT *
FROM team_projects;

-- name: ClearTeamProjectsCache :exec
TRUNCATE team_projects;
