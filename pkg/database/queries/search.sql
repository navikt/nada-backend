-- name: Search :many
SELECT
	element_id::uuid,
	element_type::text,
	ts_rank_cd(tsv_document, query) AS rank,
	ts_headline('norwegian', "description", query, 'MinWords=14, MaxWords=15, MaxFragments=2 FragmentDelimiter=" … "')::text AS excerpt
FROM
	search,
	websearch_to_tsquery('norwegian', @query) query
WHERE
	(
		CASE
			WHEN @keyword :: text != '' THEN @keyword = ANY(keywords)
			ELSE TRUE
		END
	)
	AND (
		CASE
			WHEN @query :: text != '' THEN "tsv_document" @@ query
			ELSE TRUE
		END
	)
ORDER BY rank
;
