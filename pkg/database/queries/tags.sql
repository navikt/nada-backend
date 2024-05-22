-- name: GetKeywords :many
SELECT keyword::text, count(1) as "count"
FROM (
         SELECT unnest(ds.keywords) as keyword
            FROM datasets ds
         UNION ALL
         SELECT unnest(s.keywords) as keyword
            FROM stories s
    ) keywords
GROUP BY keyword
ORDER BY keywords."count" DESC;

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

-- name: RemoveKeywordInDatasets :exec
UPDATE datasets SET keywords= array_remove(keywords, @keyword_to_remove);

-- name: ReplaceKeywordInDatasets :exec
UPDATE datasets SET keywords= array_replace(keywords, @keyword, @new_text_for_keyword);

-- name: RemoveKeywordInStories :exec
UPDATE stories SET keywords = array_remove(keywords, @keyword_to_remove);

-- name: ReplaceKeywordInStories :exec
UPDATE stories SET keywords = array_replace(keywords, @keyword, @new_text_for_keyword);
