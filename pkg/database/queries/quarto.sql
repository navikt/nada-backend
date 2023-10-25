-- name: CreateQuartoStory :one
INSERT INTO quarto_stories (
	"name",
    "creator",
	"description",
	"keywords",
	"teamkatalogen_url",
    "product_area_id",
    "team_id",
    "group"
) VALUES (
	@name,
	@creator,
	@description,
	@keywords,
	@teamkatalogen_url,
    @product_area_id,
    @team_id,
    @owner_group
)
RETURNING *;

-- name: GetQuartoStory :one
SELECT *
FROM quarto_stories
WHERE id = @id;

-- name: GetQuartoStories :many
SELECT *
FROM quarto_stories
ORDER BY last_modified DESC;

-- name: GetQuartoStoriesByIDs :many
SELECT *
FROM quarto_stories
WHERE id = ANY (@ids::uuid[])
ORDER BY last_modified DESC;

-- name: GetQuartoStoriesByProductArea :many
SELECT *
FROM quarto_stories
WHERE product_area_id = @product_area_id
ORDER BY last_modified DESC;

-- name: GetQuartoStoriesByTeam :many
SELECT *
FROM quarto_stories
WHERE team_id = @team_id
ORDER BY last_modified DESC;

-- name: UpdateQuartoStory :one
UPDATE quarto_stories
SET
	"name" = @name,
	"description" = @description,
	"keywords" = @keywords,
	"teamkatalogen_url" = @teamkatalogen_url,
    "product_area_id" = @product_area_id,
    "team_id" = @team_id,
    "group" = @owner_group
WHERE id = @id
RETURNING *;

-- name: DeleteQuartoStory :exec
DELETE FROM quarto_stories
WHERE id = @id;

-- name: GetQuartoStoriesByGroups :many
SELECT *
FROM quarto_stories
WHERE "group" = ANY (@groups::text[])
ORDER BY last_modified DESC;

-- name: ReplaceQuartoStoriesTag :exec
UPDATE quarto_stories
SET "keywords" = array_replace(keywords, @tag_to_replace, @tag_updated);