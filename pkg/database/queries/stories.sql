-- name: CreateStory :one
INSERT INTO stories (
	"name",
	"group",
	"description",
	"keywords",
	"teamkatalogen_url"
) VALUES (
	@name,
	@grp,
	@description,
	@keywords,
	@teamkatalogen_url
)
RETURNING *;

-- name: CreateStoryView :one
INSERT INTO story_views (
	"story_id",
	"sort",
	"type",
	"spec"
) VALUES (
	@story_id,
	@sort,
	@type,
	@spec
)
RETURNING *;

-- name: GetStory :one
SELECT *
FROM stories
WHERE id = @id;

-- name: GetStories :many
SELECT *
FROM stories
ORDER BY created DESC;

-- name: GetStoriesByIDs :many
SELECT *
FROM stories
WHERE id = ANY (@ids::uuid[])
ORDER BY created DESC;

-- name: GetStoryView :one
SELECT *
FROM story_views
WHERE id = @id;

-- name: GetStoryViews :many
SELECT *
FROM story_views
WHERE story_id = @story_id
ORDER BY sort ASC;

-- name: UpdateStory :one
UPDATE stories
SET
	"name" = @name,
	"group" = @grp,
	"description" = @description,
	"keywords" = @keywords,
	"teamkatalogen_url" = @teamkatalogen_url
WHERE id = @id
RETURNING *;

-- name: DeleteStory :exec
DELETE FROM stories
WHERE id = @id;

-- name: DeleteStoryViews :exec
DELETE FROM story_views
WHERE story_id = @story_id;

-- name: GetStoryToken :one
SELECT *
FROM story_tokens
WHERE story_id = @story_id;

-- name: GetStoryFromToken :one
SELECT *
FROM stories
WHERE id = (SELECT story_id FROM story_tokens WHERE token = @token);

-- name: GetStoriesByGroups :many
SELECT *
FROM stories
WHERE "group" = ANY (@groups::text[])
ORDER BY last_modified DESC;
