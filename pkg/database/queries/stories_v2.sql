-- name: GetStoriesWithTeamkatalogenByIDs :many
SELECT *
FROM story_with_teamkatalogen_view
WHERE id = ANY (@ids::uuid[])
ORDER BY last_modified DESC;