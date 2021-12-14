-- name: CreateStory :one
INSERT INTO stories (
	"name",
	"group"
) VALUES (
	@name,
	@grp
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
