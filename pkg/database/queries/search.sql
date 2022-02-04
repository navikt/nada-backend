-- name: Search :many
SELECT
	element_id::uuid,
	element_type::text,
	ts_rank_cd(tsv_document, query) AS rank,
	ts_headline('norwegian', "description", query, 'MinWords=14, MaxWords=15, MaxFragments=2 FragmentDelimiter=" … " StartSel="**" StopSel="**"')::text AS excerpt
FROM
	search,
	websearch_to_tsquery('norwegian', @query) query
WHERE
	(
		CASE
			WHEN array_length(@types::text[], 1) > 0 THEN "element_type" = ANY(@types)
			ELSE TRUE
		END
	)
	AND (
		CASE
			WHEN array_length(@keyword::text[], 1) > 0 THEN "keywords" && @keyword
			ELSE TRUE
		END
	)
	AND (
		CASE
			WHEN @query :: text != '' THEN "tsv_document" @@ query
			ELSE TRUE
		END
	)
	AND (
		CASE
			WHEN array_length(@grp::text[], 1) > 0 THEN "group" = ANY(@grp)
			ELSE TRUE
		END
	)
	AND (
		CASE
			WHEN array_length(@service::text[], 1) > 0 THEN "services" && @service
			ELSE TRUE
		END
	)
ORDER BY rank DESC, created DESC
LIMIT @lim OFFSET @offs;
;
