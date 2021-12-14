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
