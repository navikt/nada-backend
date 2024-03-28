-- name: GetStoriesWithTeamkatalogenByIDs :many
SELECT *
FROM story_with_teamkatalogen_view
WHERE id = ANY (@ids::uuid[])
ORDER BY last_modified DESC;

-- name: GetStoriesWithTeamkatalogenByGroups :many
SELECT *
FROM story_with_teamkatalogen_view
WHERE "group" = ANY (@groups::text[])
ORDER BY last_modified DESC;