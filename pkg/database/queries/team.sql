-- name: GetNadaToken :one
SELECT token
FROM nada_tokens
WHERE team = @team;

-- name: DeleteNadaToken :exec
DELETE 
FROM nada_tokens
WHERE team = @team;
