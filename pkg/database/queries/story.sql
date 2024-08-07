-- name: CreateStory :one
INSERT INTO stories (
	"name",
    "creator",
	"description",
	"keywords",
	"teamkatalogen_url",
    "team_id",
    "group"
) VALUES (
	@name,
	@creator,
	@description,
	@keywords,
	@teamkatalogen_url,
    @team_id,
    @owner_group
)
RETURNING *;

-- name: CreateStoryWithID :one
INSERT INTO stories (
    "id",
	"name",
    "creator",
	"description",
	"keywords",
	"teamkatalogen_url",
    "team_id",
    "group"
) VALUES (
    @id,
	@name,
	@creator,
	@description,
	@keywords,
	@teamkatalogen_url,
    @team_id,
    @owner_group
)
RETURNING *;

-- name: GetStory :one
SELECT *
FROM stories
WHERE id = @id;

-- name: GetStories :many
SELECT *
FROM stories
ORDER BY last_modified DESC;

-- name: GetStoriesByIDs :many
SELECT *
FROM stories
WHERE id = ANY (@ids::uuid[])
ORDER BY last_modified DESC;

-- name: GetStoriesByProductArea :many
SELECT *
FROM story_with_teamkatalogen_view
WHERE team_id = ANY(@team_id::uuid[])
ORDER BY last_modified DESC;

-- name: GetStoriesByTeam :many
SELECT *
FROM stories
WHERE team_id = @team_id
ORDER BY last_modified DESC;

-- name: GetStoriesNumberByTeam :one
SELECT COUNT(*) as "count"
FROM stories
WHERE team_id = @team_id;

-- name: UpdateStory :one
UPDATE stories
SET
	"name" = @name,
	"description" = @description,
	"keywords" = @keywords,
	"teamkatalogen_url" = @teamkatalogen_url,
    "team_id" = @team_id,
    "group" = @owner_group
WHERE id = @id
RETURNING *;

-- name: DeleteStory :exec
DELETE FROM stories
WHERE id = @id;

-- name: GetStoriesByGroups :many
SELECT *
FROM stories
WHERE "group" = ANY (@groups::text[])
ORDER BY last_modified DESC;

-- name: ReplaceStoriesTag :exec
UPDATE stories
SET "keywords" = array_replace(keywords, @tag_to_replace, @tag_updated);
