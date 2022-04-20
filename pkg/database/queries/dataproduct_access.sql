-- name: GrantAccessToDataproduct :one
INSERT INTO dataproduct_access (dataproduct_id,
                                "subject",
                                granter,
                                expires,
                                polly_id)
VALUES (@dataproduct_id,
        LOWER(@subject),
        LOWER(@granter),
        @expires,
        @polly_id)
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

-- name: ListActiveAccessToDataproduct :many
SELECT *
FROM dataproduct_access
WHERE dataproduct_id = @dataproduct_id AND revoked IS NULL AND (expires IS NULL OR expires >= NOW());

-- name: GetActiveAccessToDataproductForSubject :one
SELECT *
FROM dataproduct_access
WHERE dataproduct_id = @dataproduct_id 
AND "subject" = @subject 
AND revoked IS NULL 
AND (
  expires IS NULL 
  OR expires >= NOW()
);
