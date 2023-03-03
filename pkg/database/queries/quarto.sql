-- name: CreateQuartoStory :one
INSERT INTO quarto_stories (
	"name",
    "creator",
	"description",
	"keywords",
	"teamkatalogen_url",
    "product_area_id",
    "team_id"
) VALUES (
	@name,
	@creator,
	@description,
	@keywords,
	@teamkatalogen_url,
    @product_area_id,
    @team_id
)
RETURNING *;

-- name: GetQuartoStory :one
SELECT *
FROM quarto_stories
WHERE id = @id;

-- name: GetQuartoStories :many
SELECT *
FROM quarto_stories
ORDER BY created DESC;

-- name: GetQuartoStoriesByIDs :many
SELECT *
FROM quarto_stories
WHERE id = ANY (@ids::uuid[])
ORDER BY created DESC;

-- name: GetQuartoStoriesByProductArea :many
SELECT *
FROM quarto_stories
WHERE product_area_id = @product_area_id
ORDER BY created DESC;

-- name: GetQuartoStoriesByTeam :many
SELECT *
FROM quarto_stories
WHERE team_id = @team_id
ORDER BY created DESC;

-- name: UpdateQuartoStory :one
UPDATE quarto_stories
SET
	"name" = @name,
    "creator" = @creator,
	"description" = @description,
	"keywords" = @keywords,
	"teamkatalogen_url" = @teamkatalogen_url,
    "product_area_id" = @product_area_id,
    "team_id" = @team_id
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