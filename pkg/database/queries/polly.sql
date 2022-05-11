-- name: CreatePollyDocumentation :one
INSERT INTO polly_documentation ("external_id",
                                 "name",
                                 "url")
VALUES (@external_id,
        @name,
        @url)
RETURNING *;

-- name: GetPollyDocumentation :one
SELECT *
FROM polly_documentation
WHERE id = @id;
