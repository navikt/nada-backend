-- name: CreateAccessRequestForDataproduct :one
INSERT INTO dataproduct_access_request (dataproduct_id,
                                        "subject",
                                        "owner",
                                        "expires",
                                        polly_documentation_id)
VALUES (@dataproduct_id,
        LOWER(@subject),
        LOWER(@owner),
        @expires,
        @polly_documentation_id)
RETURNING *;

-- name: ListAccessRequestsForDataproduct :many
SELECT *
FROM dataproduct_access_request
WHERE dataproduct_id = @dataproduct_id AND status = 'pending';

-- name: ListAccessRequestsForOwner :many
SELECT *
FROM dataproduct_access_request
WHERE "owner" = ANY (@owner::text[]) AND status = 'pending';

-- name: GetAccessRequest :one
SELECT *
FROM dataproduct_access_request
WHERE id = @id;

-- name: UpdateAccessRequest :one
UPDATE dataproduct_access_request
SET owner                  = @owner,
    polly_documentation_id = @polly_documentation_id,
    expires = @expires
WHERE id = @id
RETURNING *;

-- name: DeleteAccessRequest :exec
DELETE FROM dataproduct_access_request
WHERE id = @id;

-- name: DenyAccessRequest :exec
UPDATE dataproduct_access_request
SET status = 'denied',
    granter = @granter,
    reason = @reason,
    closed = NOW()
WHERE id = @id;

-- name: ApproveAccessRequest :exec
UPDATE dataproduct_access_request
SET status = 'approved',
    granter = @granter,
    closed = NOW()
WHERE id = @id;
