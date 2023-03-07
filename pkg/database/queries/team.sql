-- name: GetNadaToken :one
SELECT token
FROM nada_tokens
WHERE team = @team;
