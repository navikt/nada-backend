-- name: CreateAccessRequestForDataproduct :one
INSERT INTO dataproduct_access_request (dataproduct_id,
                                        "subject",
                                        polly_documentation_id)
VALUES (@dataproduct_id,
        LOWER(@subject),
        @polly_documentation_id)
RETURNING *;

-- name: ListAccessRequestsForDataproduct :many
SELECT *
FROM dataproduct_access_request
WHERE dataproduct_id = @dataproduct_id;

-- name: ListAccessRequestsForUser :many
SELECT *
FROM dataproduct_access_request
WHERE subject = @subject;

-- name: GetAccessRequest :one
SELECT *
FROM dataproduct_access_request
WHERE id = @id;
