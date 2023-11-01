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
    nada_tokens
WHERE
    team = ANY (@teams :: text [])
ORDER BY
    team;

;

-- name: DeleteNadaToken :exec
DELETE FROM
    nada_tokens
WHERE
    team = @team;