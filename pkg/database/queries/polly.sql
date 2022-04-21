-- name: AddAccessDocumentation :one
INSERT INTO access_documentation ("access_id",
                                  "polly_id",
                                  "polly_name",
                                  "polly_url") 
VALUES (@access_id,
        @polly_id,
        @polly_name,
        @polly_url)
RETURNING *;

-- name: GetAccessDocumentation :one
SELECT * 
FROM access_documentation 
WHERE access_id = @access_id;
