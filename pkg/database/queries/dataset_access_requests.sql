-- name: CreateAccessRequestForDataset :one
INSERT INTO dataset_access_requests (dataset_id,
                                        "subject",
                                        "owner",
                                        "expires",
                                        polly_documentation_id)
VALUES (@dataset_id,
        @subject,
        LOWER(@owner),
        @expires,
        @polly_documentation_id)
RETURNING *;

-- name: ListAccessRequestsForDataset :many
SELECT *
FROM dataset_access_requests
WHERE dataset_id = @dataset_id AND status = 'pending'
ORDER BY created DESC;

-- name: ListAccessRequestsForOwner :many
SELECT *
FROM dataset_access_requests
WHERE "owner" = ANY (@owner::text[])
ORDER BY created DESC;

-- name: GetAccessRequest :one
SELECT *
FROM dataset_access_requests
WHERE id = @id;

-- name: UpdateAccessRequest :one
UPDATE dataset_access_requests
SET owner                  = @owner,
    polly_documentation_id = @polly_documentation_id,
    expires = @expires
WHERE id = @id
RETURNING *;

-- name: DeleteAccessRequest :exec
DELETE FROM dataset_access_requests
WHERE id = @id;

-- name: DenyAccessRequest :exec
UPDATE dataset_access_requests
SET status = 'denied',
    granter = @granter,
    reason = @reason,
    closed = NOW()
WHERE id = @id;

-- name: ApproveAccessRequest :exec
UPDATE dataset_access_requests
SET status = 'approved',
    granter = @granter,
    closed = NOW()
WHERE id = @id;
