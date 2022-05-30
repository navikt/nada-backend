-- name: CreateAccessRequestForDataset :one
INSERT INTO dataset_access_request (dataset_id,
                                        "subject",
                                        "owner",
                                        "expires",
                                        polly_documentation_id)
VALUES (@dataset_id,
        LOWER(@subject),
        LOWER(@owner),
        @expires,
        @polly_documentation_id)
RETURNING *;

-- name: ListAccessRequestsForDataset :many
SELECT *
FROM dataset_access_request
WHERE dataset_id = @dataset_id AND status = 'pending';

-- name: ListAccessRequestsForOwner :many
SELECT *
FROM dataset_access_request
WHERE "owner" = ANY (@owner::text[]);

-- name: GetAccessRequest :one
SELECT *
FROM dataset_access_request
WHERE id = @id;

-- name: UpdateAccessRequest :one
UPDATE dataset_access_request
SET owner                  = @owner,
    polly_documentation_id = @polly_documentation_id,
    expires = @expires
WHERE id = @id
RETURNING *;

-- name: DeleteAccessRequest :exec
DELETE FROM dataset_access_request
WHERE id = @id;

-- name: DenyAccessRequest :exec
UPDATE dataset_access_request
SET status = 'denied',
    granter = @granter,
    reason = @reason,
    closed = NOW()
WHERE id = @id;

-- name: ApproveAccessRequest :exec
UPDATE dataset_access_request
SET status = 'approved',
    granter = @granter,
    closed = NOW()
WHERE id = @id;
