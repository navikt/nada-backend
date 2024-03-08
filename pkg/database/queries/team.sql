-- name: GetNadaToken :one
SELECT
    token
FROM
    nada_tokens
WHERE
    team = @team;

-- name: GetNadaTokens :many
SELECT 
    *
FROM 
    nada_tokens;

-- name: GetNadaTokensForTeams :many
SELECT
    *
FROM
    nada_tokens
WHERE
    team = ANY (@teams :: text [])
ORDER BY
    team;

-- name: GetTeamFromNadaToken :one
SELECT team
FROM nada_tokens
WHERE token = @token;

-- name: RotateNadaToken :exec
UPDATE nada_tokens
SET token = gen_random_uuid()
WHERE team = @team;

-- name: DeleteNadaToken :exec
DELETE FROM
    nada_tokens
WHERE
    team = @team;