-- name: GrantAccessToDataproduct :one
INSERT INTO dataproduct_access (
	dataproduct_id,
	"subject",
	granter,
	expires
) VALUES (
	@dataproduct_id,
	@subject,
	@granter,
	@expires
)
RETURNING *;

-- name: DeleteAccessToDataproduct :exec
UPDATE dataproduct_access 
SET deleted = NOW() 
WHERE id = @id;

-- name: ListAccessToDataproduct :many
SELECT * FROM dataproduct_access
WHERE dataproduct_id = @dataproduct_id;
