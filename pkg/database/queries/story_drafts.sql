-- name: CreateStoryDraft :one
INSERT INTO story_drafts (
	"name"
) VALUES (
	@name
)
RETURNING *;

-- name: CreateStoryViewDraft :one
INSERT INTO story_view_drafts (
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

-- name: GetStoryDraft :one
SELECT *
FROM story_drafts
WHERE id = @id;

-- name: GetStoryDrafts :many
SELECT *
FROM story_drafts
ORDER BY created DESC;

-- name: GetStoryViewDraft :one
SELECT *
FROM story_view_drafts
WHERE id = @id;

-- name: GetStoryViewDrafts :many
SELECT *
FROM story_view_drafts
WHERE story_id = @story_id
ORDER BY sort ASC;

-- name: DeleteStoryDraft :exec
DELETE FROM story_drafts
WHERE id = @id;

-- name: DeleteStoryViewDraft :exec
DELETE FROM story_view_drafts
WHERE story_id = @story_id;

-- name: CleanupStoryDrafts :exec
DELETE FROM story_drafts
WHERE created < (NOW() - '7 day'::interval);
