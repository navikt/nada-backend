-- name: CreateSession :exec
INSERT INTO sessions (
	"token",
	"email",
	"name",
	"expires"
) VALUES (
	@token,
	@email,
	@name,
	@expires
);

-- name: GetSession :one
SELECT *
FROM sessions
WHERE token = @token
AND expires > now();
