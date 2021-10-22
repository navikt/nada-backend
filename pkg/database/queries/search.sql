-- name: Search :many
SELECT
	element_id :: uuid,
	element_type :: text,
	ts_rank_cd(tsv_document, query)
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
			WHEN @query :: text != '' THEN "tsv_document" @ @ query
			ELSE TRUE
		END
	);
