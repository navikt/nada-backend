-- name: GetDashboard :one
SELECT *
FROM "dashboards"
WHERE id = @id;
