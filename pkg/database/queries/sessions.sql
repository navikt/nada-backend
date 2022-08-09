-- name: CreateSession :exec
INSERT INTO sessions (
	"token",
	"access_token",
	"email",
	"name",
	"expires"
) VALUES (
	@token,
	@access_token,
	LOWER(@email),
	@name,
	@expires
);

-- name: GetSession :one
SELECT *
FROM sessions
WHERE token = @token
AND expires > now();

-- name: DeleteSession :exec
DELETE
FROM sessions
WHERE token = @token;