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

-- name: GetStoryDrafts :many
SELECT *
FROM story_drafts
ORDER BY created DESC;

-- name: GetStoryViewDrafts :many
SELECT *
FROM story_view_drafts
WHERE story_id = @story_id
ORDER BY sort ASC;
