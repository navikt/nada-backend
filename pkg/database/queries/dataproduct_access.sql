-- name: GrantAccessToDataproduct :one
INSERT INTO dataproduct_access (dataproduct_id,
                                "subject",
                                granter,
                                expires)
VALUES (@dataproduct_id,
        @subject,
        @granter,
        @expires)
RETURNING *;

-- name: RevokeAccessToDataproduct :exec
UPDATE dataproduct_access
SET revoked = NOW()
WHERE id = @id;

-- name: ListUnrevokedExpiredAccessEntries :many
SELECT *
FROM dataproduct_access
WHERE revoked IS NULL
  AND expires < NOW();

-- name: ListAccessToDataproduct :many
SELECT *
FROM dataproduct_access
WHERE dataproduct_id = @dataproduct_id;

-- name: GetAccessToDataproduct :one
SELECT *
FROM dataproduct_access
WHERE id = @id;
