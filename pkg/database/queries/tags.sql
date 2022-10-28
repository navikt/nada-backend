-- name: GetTags :many
SELECT * FROM tags;

-- name: GetTag :one
SELECT * FROM tags WHERE id=@id;

-- name: GetTagByPhrase :one
SELECT * FROM tags WHERE phrase=@phrase;

-- name: CreateTagIfNotExist :exec
INSERT INTO tags(phrase) VALUES (@phrase) ON CONFLICT DO NOTHING;

-- name: UpdateTag :exec
UPDATE tags SET phrase = @new_phrase where phrase = @old_phrase;